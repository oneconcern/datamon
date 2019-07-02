// Copyright © 2018 One Concern

package core

import (
	"context"
	"fmt"
	"sync"

	"github.com/oneconcern/datamon/pkg/cafs"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

const (
	fileDownloadsPerConcurrentChunks = 3
)

func unpackBundleDescriptor(ctx context.Context, bundle *Bundle) error {

	bundleDescriptorBuffer, err := storage.ReadTee(ctx,
		bundle.MetaStore, model.GetArchivePathToBundle(bundle.RepoID, bundle.BundleID),
		bundle.ConsumableStore, model.GetConsumablePathToBundle(bundle.BundleID))
	if err != nil {
		return err
	}

	// Unmarshal the file
	err = yaml.Unmarshal(bundleDescriptorBuffer, &bundle.BundleDescriptor)
	if err != nil {
		return err
	}
	return nil
}

type downloadBundleFileListChans struct {
	bundleEntries      chan<- model.BundleEntries
	error              chan<- error
	done               <-chan struct{}
	doneOk             chan<- struct{}
	concurrencyControl <-chan struct{}
}

func downloadBundleFileListFile(ctx context.Context, bundle *Bundle,
	chans downloadBundleFileListChans,
	i uint64,
) {
	var bundleEntries model.BundleEntries

	sendErr := func(err error) {
		select {
		case chans.error <- err:
		case <-chans.done:
		}
	}
	defer func() {
		<-chans.concurrencyControl
	}()

	bundle.l.Info("downloading bundle entry",
		zap.Uint64("curr entry", i),
		zap.Uint64("tot entries", bundle.BundleDescriptor.BundleEntriesFileCount),
	)
	bundleEntriesBuffer, err := storage.ReadTee(ctx,
		bundle.MetaStore, model.GetArchivePathToBundleFileList(bundle.RepoID, bundle.BundleID, i),
		bundle.ConsumableStore, model.GetConsumablePathToBundleFileList(bundle.BundleID, i))
	if err != nil {
		sendErr(err)
		return
	}
	err = yaml.Unmarshal(bundleEntriesBuffer, &bundleEntries)
	if err != nil {
		sendErr(err)
		return
	}
	select {
	case chans.bundleEntries <- bundleEntries:
	case <-chans.done:
	}

}

func downloadBundleFileList(ctx context.Context, bundle *Bundle,
	chans downloadBundleFileListChans,
) {
	var i uint64
	// todo: percolate constant to ui
	concurrencyControl := make(chan struct{}, 8)
	chans.concurrencyControl = concurrencyControl
	for i = 0; i < bundle.BundleDescriptor.BundleEntriesFileCount; i++ {
		concurrencyControl <- struct{}{}
		go downloadBundleFileListFile(ctx, bundle, chans, i)
	}
	for i := 0; i < cap(concurrencyControl); i++ {
		concurrencyControl <- struct{}{}
	}
	chans.doneOk <- struct{}{}
}

// ??? orig impl guarantees returning the entries in a particular order.
//    .. and this could as well, provided the total number of bundle entires doesn't exceed uint64 max.
// for now, foregoing ordering in order not to have limit on bundle entries based on num bits.
func unpackBundleFileList(ctx context.Context, bundle *Bundle) error {

	// todo: pass  param as in implUpload()
	var bundleEntriesPerFile uint64 = defaultBundleEntriesPerFile

	bundleEntriesC := make(chan model.BundleEntries)
	errorC := make(chan error)
	doneC := make(chan struct{})
	doneOkC := make(chan struct{})

	defer close(doneC)

	go downloadBundleFileList(ctx, bundle, downloadBundleFileListChans{
		bundleEntries: bundleEntriesC,
		error:         errorC,
		done:          doneC,
		doneOk:        doneOkC,
	})

	// prealloc
	bundle.BundleEntries = make([]model.BundleEntry, 0,
		bundle.BundleDescriptor.BundleEntriesFileCount*bundleEntriesPerFile)
	var gotDoneSignal bool
	for !gotDoneSignal {
		select {
		case bundleEntries := <-bundleEntriesC:
			bundle.BundleEntries = append(bundle.BundleEntries, bundleEntries.BundleEntries...)
		case err := <-errorC:
			bundle.l.Error("Unpack bundle filelist failed", zap.Error(err))
			return err
		case <-doneOkC:
			gotDoneSignal = true
		}
	}
	return nil
}

func unpackBundleFileListOrig(ctx context.Context, bundle *Bundle) error { // nolint: deadcode, unused
	// Download the files json
	var i uint64
	for i = 0; i < bundle.BundleDescriptor.BundleEntriesFileCount; i++ {
		bundle.l.Info("downloading bundle entry",
			zap.Uint64("curr entry", i),
			zap.Uint64("tot entries", bundle.BundleDescriptor.BundleEntriesFileCount),
		)
		bundleEntriesBuffer, err := storage.ReadTee(ctx,
			bundle.MetaStore, model.GetArchivePathToBundleFileList(bundle.RepoID, bundle.BundleID, i),
			bundle.ConsumableStore, model.GetConsumablePathToBundleFileList(bundle.BundleID, i))
		if err != nil {
			return err
		}
		var bundleEntries model.BundleEntries
		err = yaml.Unmarshal(bundleEntriesBuffer, &bundleEntries)
		if err != nil {
			return err
		}
		bundle.BundleEntries = append(bundle.BundleEntries, bundleEntries.BundleEntries...)
	}
	// Link the file
	return nil
}

type errorHit struct {
	error error
	file  string
}

func unpackDataFiles(ctx context.Context, bundle *Bundle, file string) error {
	ls := bundle.BundleDescriptor.LeafSize
	fs, err := cafs.New(
		cafs.LeafSize(ls),
		cafs.LeafTruncation(bundle.BundleDescriptor.Version < 1),
		cafs.Backend(bundle.BlobStore),
		cafs.ReaderConcurrentChunkWrites(bundle.concurrentFileDownloads/fileDownloadsPerConcurrentChunks),
	)

	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	errC := make(chan errorHit, len(bundle.BundleEntries))
	concurrentFileDownloads := bundle.concurrentFileDownloads
	if concurrentFileDownloads < 1 {
		concurrentFileDownloads = 1
	}
	concurrencyControl := make(chan struct{}, concurrentFileDownloads)
	fmt.Printf("Downloading %d files\n", len(bundle.BundleEntries))
	for _, b := range bundle.BundleEntries {
		if file != "" && file != b.NameWithPath {
			continue
		}
		wg.Add(1)
		go func(bundleEntry model.BundleEntry, cc chan struct{}) {
			cc <- struct{}{}
			defer func() {
				<-cc
			}()

			defer wg.Done()
			fmt.Println("started " + bundleEntry.NameWithPath)
			key, err := cafs.KeyFromString(bundleEntry.Hash)
			if err != nil {
				errC <- errorHit{
					err,
					bundleEntry.NameWithPath,
				}
				return
			}
			reader, err := fs.Get(ctx, key)
			if err != nil {
				errC <- errorHit{
					err,
					bundleEntry.NameWithPath,
				}
				return
			}
			err = bundle.ConsumableStore.Put(ctx, bundleEntry.NameWithPath, reader, storage.IfNotPresent)
			if err != nil {
				fmt.Printf("Failed to download %s error %s", bundleEntry.NameWithPath, err)
				errC <- errorHit{
					err,
					bundleEntry.NameWithPath,
				}
				return
			}
			fmt.Printf("downloaded %s\n", bundleEntry.NameWithPath)
		}(b, concurrencyControl)
	}
	wg.Wait()
	select {
	case eh := <-errC:
		return eh.error
	default:
		return nil
	}
}
