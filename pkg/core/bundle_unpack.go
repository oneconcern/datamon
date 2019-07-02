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

func unpackBundleFileList(ctx context.Context, bundle *Bundle) error {
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
