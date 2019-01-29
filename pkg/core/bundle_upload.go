// Copyright © 2018 One Concern

package core

import (
	"bytes"
	"context"
	"log"

	"gopkg.in/yaml.v2"

	"github.com/oneconcern/datamon/pkg/cafs"
	"github.com/oneconcern/datamon/pkg/model"
)

const (
	bundleEntriesPerFile = 1000
)

func uploadBundle(ctx context.Context, consumableBundle Bundle, archiveBundle *Bundle) error {
	// Walk the entire tree
	// TODO: #53 handle large file count
	token := ""
	cafsArchive, err := cafs.New(
		cafs.LeafSize(archiveBundle.bundleDescriptor.LeafSize),
		cafs.Backend(archiveBundle.store),
		cafs.Prefix(model.GetArchivePathBlobPrefix()),
	)

	fileList := model.BundleEntries{}
	var bundleEntriesIndex uint
	var totalWritten uint64
	var filesInBundle uint64
	// Upload the files and the bundle list
	err = archiveBundle.InitializeBundleID()
	if err != nil {
		return err
	}

	for  {
		files,token, err := consumableBundle.store.Keys(ctx, token)
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

			fileReader, err := consumableBundle.store.Get(ctx, file)
			if err != nil {
				return err
			}

			written, key, keys, err := cafsArchive.Put(ctx, fileReader)
			totalWritten = totalWritten + uint64(written)
			filesInBundle++
			if err != nil {
				return err
			}

			log.Printf("Uploaded root key %s & %d bytes for keys for %s", key.String(), len(keys), file)

			fileList.BundleEntries = append(fileList.BundleEntries, model.BundleEntry{
				Hash:         key.String(),
				NameWithPath: file,
				FileMode:     0, // #TODO: #35 file mode support
				Size:         uint64(written)})

			// Write the bundle entry file if reached max or the last one
			if index == len(files)-1 || bundleEntriesIndex == bundleEntriesPerFile {
				buffer, err := yaml.Marshal(fileList)
				if err != nil {
					return err
				}
				err = archiveBundle.store.Put(ctx,
					model.GetArchivePathToBundleFileList(
						archiveBundle.repoID,
						archiveBundle.bundleID,
						archiveBundle.bundleDescriptor.BundleEntriesFileCount),
					bytes.NewReader(buffer))
				if err != nil {
					return err
				}
				archiveBundle.bundleDescriptor.BundleEntriesFileCount++
			}
		}

		err = uploadBundleDescriptor(ctx, archiveBundle)
		if err != nil {
			return err
		}

		if token == "" {
			break
		}

		log.Printf("Uploaded bundle id:%s with %d files and %d bytes written", archiveBundle.bundleID,
			filesInBundle, totalWritten)
	}

	return nil
}

func uploadBundleDescriptor(ctx context.Context, bundle *Bundle) error {

	buffer, err := yaml.Marshal(bundle.bundleDescriptor)
	if err != nil {
		return err
	}

	err = bundle.store.Put(ctx,
		model.GetArchivePathToBundle(bundle.repoID, bundle.bundleID),
		bytes.NewReader(buffer))
	if err != nil {
		return err
	}
	return nil
}
