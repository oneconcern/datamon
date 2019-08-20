// Copyright © 2018 One Concern

package core

import (
	"context"
	"fmt"

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

type bundleEntriesRes struct {
	bundleEntries model.BundleEntries
	idx           uint64
}

type downloadBundleFileListChans struct {
	bundleEntries      chan<- bundleEntriesRes
	error              chan<- error
	doneOk             chan<- struct{}
	concurrencyControl <-chan struct{}
}

func downloadBundleFileListFile(ctx context.Context, bundle *Bundle,
	chans downloadBundleFileListChans,
	i uint64,
) {
	var bundleEntries model.BundleEntries

	sendErr := func(err error) {
		chans.error <- err
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
	chans.bundleEntries <- bundleEntriesRes{bundleEntries: bundleEntries, idx: i}

}

func downloadBundleFileList(ctx context.Context, bundle *Bundle,
	chans downloadBundleFileListChans,
) {
	var i uint64
	concurrencyControl := make(chan struct{}, bundle.concurrentFilelistDownloads)
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

func unpackBundleFileList(ctx context.Context, bundle *Bundle, bundleEntriesPerFile uint) error {

	bundleEntriesC := make(chan bundleEntriesRes)
	errorC := make(chan error)
	doneC := make(chan struct{})
	doneOkC := make(chan struct{})

	defer close(doneC)

	go downloadBundleFileList(ctx, bundle, downloadBundleFileListChans{
		bundleEntries: bundleEntriesC,
		error:         errorC,
		doneOk:        doneOkC,
	})

	// prealloc
	maxBundleEntries := bundle.BundleDescriptor.BundleEntriesFileCount * uint64(bundleEntriesPerFile)
	bundle.l.Info("preallocating bundle entries",
		zap.Uint64("max entries", maxBundleEntries),
	)
	bundle.BundleEntries = make([]model.BundleEntry, maxBundleEntries)

	var gotDoneSignal bool
	for !gotDoneSignal {
		select {
		case res := <-bundleEntriesC:
			startIdx := int(res.idx) * int(bundleEntriesPerFile)
			copy(bundle.BundleEntries[startIdx:], res.bundleEntries.BundleEntries)
			if res.idx+1 == bundle.BundleDescriptor.BundleEntriesFileCount {
				missingEntries := int(bundleEntriesPerFile) - len(res.bundleEntries.BundleEntries)
				if missingEntries < 0 {
					return fmt.Errorf("%v is greater than expected number of bundler entries %v",
						len(res.bundleEntries.BundleEntries), bundleEntriesPerFile)
				}
				bundle.BundleEntries = bundle.BundleEntries[:len(bundle.BundleEntries)-missingEntries]
			} else if uint(len(res.bundleEntries.BundleEntries)) != bundleEntriesPerFile {
				return fmt.Errorf("%v is not expected number of bundler entries %v",
					len(res.bundleEntries.BundleEntries), bundleEntriesPerFile)
			}
		case err := <-errorC:
			bundle.l.Error("Unpack bundle filelist failed", zap.Error(err))
			return err
		case <-doneOkC:
			gotDoneSignal = true
		}
	}
	return nil
}

type errorHit struct {
	error error
	file  string
}

type downloadBundleChans struct {
	error              chan<- errorHit
	doneOk             chan<- struct{}
	concurrencyControl <-chan struct{}
}

func downloadBundleEntrySync(ctx context.Context, bundleEntry model.BundleEntry,
	bundle *Bundle,
	fs cafs.Fs) error {
	bundle.l.Info("starting bundle entry download",
		zap.String("name", bundleEntry.NameWithPath))
	key, err := cafs.KeyFromString(bundleEntry.Hash)
	if err != nil {
		return err
	}
	reader, err := fs.Get(ctx, key)
	if err != nil {
		return err
	}
	err = bundle.ConsumableStore.Put(ctx, bundleEntry.NameWithPath, reader, storage.IfNotPresent)
	if err != nil {
		bundle.l.Error("Failed to download bundle entry: put to store",
			zap.String("name", bundleEntry.NameWithPath),
			zap.Error(err))
		return err
	}
	bundle.l.Info("downloaded bundle entry",
		zap.String("name", bundleEntry.NameWithPath))
	return nil
}

func downloadBundleEntry(ctx context.Context, bundleEntry model.BundleEntry,
	bundle *Bundle,
	fs cafs.Fs,
	chans downloadBundleChans) {
	defer func() {
		<-chans.concurrencyControl
	}()
	reportError := func(err error) {
		chans.error <- errorHit{
			err,
			bundleEntry.NameWithPath,
		}
	}
	err := downloadBundleEntrySync(ctx, bundleEntry, bundle, fs)
	if err != nil {
		reportError(err)
		return
	}
}

func downloadBundleEntries(ctx context.Context, bundle *Bundle,
	selectionPredicate func(string) (bool, error),
	fs cafs.Fs,
	chans downloadBundleChans) {
	var selectionPredicateOk bool
	var err error
	reportError := func(err error) {
		chans.error <- errorHit{
			err,
			"",
		}
	}
	concurrentFileDownloads := bundle.concurrentFileDownloads
	if concurrentFileDownloads < 1 {
		concurrentFileDownloads = 1
	}
	concurrencyControl := make(chan struct{}, concurrentFileDownloads)
	chans.concurrencyControl = concurrencyControl
	bundle.l.Info("downloading bundle entries",
		zap.Int("num", len(bundle.BundleEntries)))
	for _, b := range bundle.BundleEntries {
		if selectionPredicate != nil {
			selectionPredicateOk, err = selectionPredicate(b.NameWithPath)
			if err != nil {
				reportError(err)
				break
			}
		}
		if selectionPredicate == nil || selectionPredicateOk {
			concurrencyControl <- struct{}{}
			go downloadBundleEntry(ctx, b, bundle, fs, chans)
		}
	}
	for i := 0; i < cap(concurrencyControl); i++ {
		concurrencyControl <- struct{}{}
	}
	chans.doneOk <- struct{}{}
}

func unpackDataFiles(ctx context.Context, bundle *Bundle, selectionPredicate func(string) (bool, error)) error {
	fs, err := cafs.New(
		cafs.LeafSize(bundle.BundleDescriptor.LeafSize),
		cafs.LeafTruncation(bundle.BundleDescriptor.Version < 1),
		cafs.Backend(bundle.BlobStore),
		cafs.ReaderConcurrentChunkWrites(bundle.concurrentFileDownloads/fileDownloadsPerConcurrentChunks),
	)
	if err != nil {
		return err
	}
	errC := make(chan errorHit)
	doneOkC := make(chan struct{})
	go downloadBundleEntries(
		ctx,
		bundle,
		selectionPredicate,
		fs,
		downloadBundleChans{
			error:  errC,
			doneOk: doneOkC,
		})
	select {
	case eh := <-errC:
		return eh.error
	case <-doneOkC:
		return nil
	}
}

func unpackDataFile(ctx context.Context, bundle *Bundle, file string) error {
	fs, err := cafs.New(
		cafs.LeafSize(bundle.BundleDescriptor.LeafSize),
		cafs.LeafTruncation(bundle.BundleDescriptor.Version < 1),
		cafs.Backend(bundle.BlobStore),
		cafs.ReaderConcurrentChunkWrites(bundle.concurrentFileDownloads/fileDownloadsPerConcurrentChunks),
	)
	if err != nil {
		return err
	}
	bundle.l.Info("downloading bundle file",
		zap.String("name", file))
	var foundFile bool
	for _, b := range bundle.BundleEntries {
		if file != b.NameWithPath {
			continue
		}
		foundFile = true
		err = downloadBundleEntrySync(ctx, b, bundle, fs)
		if err != nil {
			return err
		}
	}
	if !foundFile {
		return fmt.Errorf("didn't find file '%v' in bundle '%v'", file, bundle.BundleID)
	}
	return nil
}
