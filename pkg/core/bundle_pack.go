// Copyright © 2018 One Concern

package core

import (
	"bytes"
	"context"
	"fmt"
	"hash/crc32"
	"io"
	"log"

	"github.com/oneconcern/datamon/pkg/storage"

	"gopkg.in/yaml.v2"

	"github.com/oneconcern/datamon/pkg/cafs"
	"github.com/oneconcern/datamon/pkg/model"
)

const (
	defaultBundleEntriesPerFile = 1000
)

type filePacked struct {
	hash      string
	name      string
	keys      []byte
	size      uint64
	duplicate bool
}

func uploadBundleEntriesFileList(ctx context.Context, bundle *Bundle, fileList []model.BundleEntry) error {
	buffer, err := yaml.Marshal(model.BundleEntries{
		BundleEntries: fileList,
	})
	if err != nil {
		return err
	}
	msCRC, ok := bundle.MetaStore.(storage.StoreCRC)
	if ok {
		crc := crc32.Checksum(buffer, crc32.MakeTable(crc32.Castagnoli))
		err = msCRC.PutCRC(ctx,
			model.GetArchivePathToBundleFileList(
				bundle.RepoID,
				bundle.BundleID,
				bundle.BundleDescriptor.BundleEntriesFileCount),
			bytes.NewReader(buffer), storage.IfNotPresent, crc)
	} else {
		err = bundle.MetaStore.Put(ctx,
			model.GetArchivePathToBundleFileList(
				bundle.RepoID,
				bundle.BundleID,
				bundle.BundleDescriptor.BundleEntriesFileCount),
			bytes.NewReader(buffer), storage.IfNotPresent)
	}
	if err != nil {
		return err
	}
	bundle.BundleDescriptor.BundleEntriesFileCount++
	return nil
}

func uploadBundle(ctx context.Context, bundle *Bundle, bundleEntriesPerFile uint, getKeys func() ([]string, error)) error {
	// Walk the entire tree
	// TODO: #53 handle large file count
	files, err := getKeys()
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

	fileList := make([]model.BundleEntry, 0)
	var firstUnuploadBundleEntryIndex uint
	// Upload the files and the bundle list
	err = bundle.InitializeBundleID()
	if err != nil {
		return err
	}

	fC := make(chan filePacked, len(files))
	eC := make(chan errorHit, len(files))
	cC := make(chan struct{}, 20)
	var count int64
	for _, file := range files {
		// Check to see if the file is to be skipped.
		if model.IsGeneratedFile(file) {
			continue
		}

		var fileReader io.ReadCloser
		fileReader, err = bundle.ConsumableStore.Get(ctx, file)
		if err != nil {
			return err
		}
		count++
		cC <- struct{}{}
		go func(file string) {
			defer func() {
				<-cC
			}()
			written, key, keys, duplicate, e := cafsArchive.Put(ctx, fileReader)
			if e != nil {
				eC <- errorHit{
					error: e,
					file:  file,
				}
				return
			}

			fC <- filePacked{
				hash:      key.String(),
				keys:      keys,
				name:      file,
				size:      uint64(written),
				duplicate: duplicate,
			}
		}(file)
	}
	for count > 0 {
		select {
		case f := <-fC:
			log.Printf("Uploaded file:%s, duplicate:%t, key:%s, keys:%d", f.name, f.duplicate, f.hash, len(f.keys))

			count--

			fileList = append(fileList, model.BundleEntry{
				Hash:         f.hash,
				NameWithPath: f.name,
				FileMode:     0, // #TODO: #35 file mode support
				Size:         f.size})

			// Write the bundle entry file if reached max or the last one
			if count == 0 || uint(len(fileList))%bundleEntriesPerFile == 0 {
				err = uploadBundleEntriesFileList(ctx, bundle, fileList[firstUnuploadBundleEntryIndex:])
				if err != nil {
					fmt.Printf("Bundle upload failed.  Failed to upload bundle entries list %v", err)
					return err
				}
				firstUnuploadBundleEntryIndex = uint(len(fileList))
			}
		case e := <-eC:
			count--
			fmt.Printf("Bundle upload failed. Failed to upload file %s err: %s", e.file, e.error)
			return e.error
		}
	}
	err = uploadBundleDescriptor(ctx, bundle)
	if err != nil {
		return err
	}
	log.Printf("Uploaded bundle id:%s ", bundle.BundleID)
	return nil
}

func uploadBundleDescriptor(ctx context.Context, bundle *Bundle) error {

	buffer, err := yaml.Marshal(bundle.BundleDescriptor)
	if err != nil {
		return err
	}
	msCRC, ok := bundle.MetaStore.(storage.StoreCRC)
	if ok {
		crc := crc32.Checksum(buffer, crc32.MakeTable(crc32.Castagnoli))
		err = msCRC.PutCRC(ctx,
			model.GetArchivePathToBundle(bundle.RepoID, bundle.BundleID),
			bytes.NewReader(buffer), storage.IfNotPresent, crc)

	} else {
		err = bundle.MetaStore.Put(ctx,
			model.GetArchivePathToBundle(bundle.RepoID, bundle.BundleID),
			bytes.NewReader(buffer), storage.IfNotPresent)
	}
	if err != nil {
		return err
	}
	return nil
}
