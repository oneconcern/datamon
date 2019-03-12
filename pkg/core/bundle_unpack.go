// Copyright © 2018 One Concern

package core

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/oneconcern/datamon/pkg/cafs"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	"gopkg.in/yaml.v2"
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

func unpackDataFiles(ctx context.Context, bundle *Bundle) error {
	fs, err := cafs.New(
		cafs.LeafSize(bundle.BundleDescriptor.LeafSize),
		cafs.Backend(bundle.BlobStore),
	)

	if err != nil {
		return err
	}
	var count int64
	errC := make(chan errorHit, len(bundle.BundleEntries))
	fC := make(chan string, len(bundle.BundleEntries))
	for _, b := range bundle.BundleEntries {
		fmt.Println("started " + b.NameWithPath)
		go func(bundleEntry model.BundleEntry) {
			atomic.AddInt64(&count, 1)
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
			fC <- bundleEntry.NameWithPath
		}(b)
	}
	for {
		if atomic.LoadInt64(&count) == 0 {
			break
		}
		select {
		case eh := <-errC:
			return eh.error
		case _ = <-fC:
			atomic.AddInt64(&count, -1)
		default:
			continue
		}
	}
	return nil
}
