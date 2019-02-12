// Copyright © 2018 One Concern

package core

import (
	"bytes"
	"context"
	"io"
	"log"

	"gopkg.in/yaml.v2"

	"github.com/oneconcern/datamon/pkg/cafs"
	"github.com/oneconcern/datamon/pkg/model"
)

const (
	bundleEntriesPerFile = 1000
)

func uploadBundle(ctx context.Context, bundle *Bundle) error {
	// Walk the entire tree
	// TODO: #53 handle large file count
	files, err := bundle.ConsumableStore.Keys(ctx)
	if err != nil {
		return err
	}
	cafsArchive, err := cafs.New(
		cafs.LeafSize(bundle.BundleDescriptor.LeafSize),
		cafs.Backend(bundle.BlobStore),
	)
	if err != nil {
		return err
	}

	fileList := model.BundleEntries{}
	var bundleEntriesIndex uint
	var totalWritten uint64
	var filesInBundle uint64
	// Upload the files and the bundle list
	err = bundle.InitializeBundleID()
	if err != nil {
		return err
	}
	for index, file := range files {
		// Check to see if the file is to be skipped.
		if model.IsGeneratedFile(file) {
			continue
		}
		bundleEntriesIndex++
		if bundleEntriesIndex == bundleEntriesPerFile {
			bundleEntriesIndex = 0
		}

		var fileReader io.ReadCloser
		fileReader, err = bundle.ConsumableStore.Get(ctx, file)
		if err != nil {
			return err
		}

		written, key, keys, e := cafsArchive.Put(ctx, fileReader)
		totalWritten += uint64(written)
		filesInBundle++
		if e != nil {
			return e
		}

		log.Printf("Uploaded root key %s & %d bytes for keys for %s", key.String(), len(keys), file)

		fileList.BundleEntries = append(fileList.BundleEntries, model.BundleEntry{
			Hash:         key.String(),
			NameWithPath: file,
			FileMode:     0, // #TODO: #35 file mode support
			Size:         uint64(written)})

		// Write the bundle entry file if reached max or the last one
		if index == len(files)-1 || bundleEntriesIndex == bundleEntriesPerFile {
			buffer, e := yaml.Marshal(fileList)
			if e != nil {
				return e
			}
			err = bundle.ArchiveStore.Put(ctx,
				model.GetArchivePathToBundleFileList(
					bundle.RepoID,
					bundle.BundleID,
					bundle.BundleDescriptor.BundleEntriesFileCount),
				bytes.NewReader(buffer))
			if err != nil {
				return err
			}
			bundle.BundleDescriptor.BundleEntriesFileCount++
		}
	}

	err = uploadBundleDescriptor(ctx, bundle)
	if err != nil {
		return err
	}

	log.Printf("Uploaded bundle id:%s with %d files and %d bytes written", bundle.BundleID,
		filesInBundle, totalWritten)

	return nil
}

func uploadBundleDescriptor(ctx context.Context, bundle *Bundle) error {

	buffer, err := yaml.Marshal(bundle.BundleDescriptor)
	if err != nil {
		return err
	}

	err = bundle.ArchiveStore.Put(ctx,
		model.GetArchivePathToBundle(bundle.RepoID, bundle.BundleID),
		bytes.NewReader(buffer))
	if err != nil {
		return err
	}
	return nil
}
