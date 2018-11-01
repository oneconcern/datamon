// Copyright © 2018 One Concern

package core

import (
	"context"
	"github.com/oneconcern/datamon/pkg/cafs"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	"gopkg.in/yaml.v2"
)

func unpackBundleDescriptor(ctx context.Context, archiveBundle *ArchiveBundle, consumableBundle ConsumableBundle) error {

	bundleDescriptorBuffer, err := storage.ReadTee(ctx,
		archiveBundle.store, model.GetArchivePathToBundle(archiveBundle.repoId, archiveBundle.bundleId),
		consumableBundle.Store,  model.GetConsumablePathToBundle(archiveBundle.bundleId))
	if err != nil {
		return err
	}

	// Unmarshal the file
	err = yaml.Unmarshal(bundleDescriptorBuffer, &archiveBundle.bundleDescriptor)
	if err != nil {
		return err
	}
	return nil
}

func unpackBundleFileList(ctx context.Context, archiveBundle *ArchiveBundle, consumableBundle ConsumableBundle) error {
	// Download the files json
	var i int64
	for i = 0; i < archiveBundle.bundleDescriptor.EntryFilesCount; i++ {
		bundleEntriesBuffer, err := storage.ReadTee(ctx,
			archiveBundle.store, model.GetArchivePathToBundleFileList(archiveBundle.repoId, archiveBundle.bundleId, i),
			consumableBundle.Store, model.GetConsumablePathToBundleFileList(archiveBundle.bundleId, i))
		if err != nil {
			return err
		}
		var bundleEntries model.BundleEntries
		err = yaml.Unmarshal(bundleEntriesBuffer, &bundleEntries)
		if err != nil {
			return err
		}
		archiveBundle.bundleEntries = append(archiveBundle.bundleEntries, bundleEntries.BundleEntries...)
	}
	// Link the file
	return nil
}

func unpackDataFiles(ctx context.Context, archiveBundle *ArchiveBundle, consumableBundle ConsumableBundle) error {
	fs, err := cafs.New(
		cafs.LeafSize(archiveBundle.bundleDescriptor.LeafSize),
		cafs.Backend(archiveBundle.store),
		cafs.Prefix(model.GetArchivePathBlobPrefix()),
	)
	if err != nil {
		return err
	}
	for _, bundleEntry := range archiveBundle.bundleEntries {
		key, err := cafs.KeyFromString(bundleEntry.Hash)
		if err != nil {
			return err
		}
		reader, err := fs.Get(ctx, key)
		err = consumableBundle.Store.Put(ctx, bundleEntry.NameWithPath, reader)
		if err != nil {
			return err
		}
	}
	return nil
}
