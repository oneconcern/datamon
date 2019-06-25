// Copyright © 2018 One Concern

package core

import (
	"bytes"
	"context"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"

	"go.uber.org/zap"

	"github.com/oneconcern/datamon/pkg/storage"

	"gopkg.in/yaml.v2"

	"github.com/oneconcern/datamon/pkg/cafs"
	"github.com/oneconcern/datamon/pkg/model"
)

const (
	defaultBundleEntriesPerFile = 1000
	fileUploadsPerFlush         = 4
)

type filePacked struct {
	hash      string
	name      string
	keys      []byte
	size      uint64
	duplicate bool
}

func filePacked2BundleEntry(packedFile filePacked) model.BundleEntry {
	return model.BundleEntry{
		Hash:         packedFile.hash,
		NameWithPath: packedFile.name,
		FileMode:     0, // #TODO: #35 file mode support
		Size:         packedFile.size,
	}
}

type uploadBundleChans struct {
	// recv data from goroutines about uploaded files
	filePacked chan<- filePacked
	error      chan<- errorHit
	// broadcast to all goroutines not to block by closing this channel
	done <-chan struct{}
	// signal file upload goroutines done by writing to this channel
	doneOk             chan<- struct{}
	concurrencyControl <-chan struct{}
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

func (b *Bundle) skipFile(file string) bool {
	exist, err := b.ConsumableStore.Has(context.Background(), file)
	if err != nil {
		b.l.Error("could not check if file exists",
			zap.String("file", file),
			zap.String("repo", b.RepoID),
			zap.String("bundleID", b.BundleID))
		exist = true // Code will decide later how to handle this file
	}
	return model.IsGeneratedFile(file) || (b.SkipOnError && !exist)
}

func uploadBundleFile(
	ctx context.Context,
	file string,
	cafsArchive cafs.Fs,
	fileReader io.Reader,
	chans uploadBundleChans) {

	defer func() {
		<-chans.concurrencyControl
	}()
	putRes, e := cafsArchive.Put(ctx, fileReader)
	if e != nil {
		select {
		case chans.error <- errorHit{
			error: e,
			file:  file,
		}:
		case <-chans.done:
		}
		return
	}

	select {
	case chans.filePacked <- filePacked{
		hash:      putRes.Key.String(),
		keys:      putRes.Keys,
		name:      file,
		size:      uint64(putRes.Written),
		duplicate: putRes.Found,
	}:
	case <-chans.done:
	}
}

func uploadBundleFiles(
	ctx context.Context,
	bundle *Bundle,
	files []string,
	cafsArchive cafs.Fs,
	chans uploadBundleChans) {
	concurrencyControl := make(chan struct{}, bundle.concurrentFileUploads)
	chans.concurrencyControl = concurrencyControl
	for _, file := range files {
		// Check to see if the file is to be skipped.
		if bundle.skipFile(file) {
			bundle.l.Info("skipping file",
				zap.String("file", file),
				zap.String("repo", bundle.RepoID),
				zap.String("bundleID", bundle.BundleID),
			)
			continue
		}
		fileReader, err := bundle.ConsumableStore.Get(ctx, file)
		if err != nil {
			if bundle.SkipOnError {
				bundle.l.Info("skipping file",
					zap.String("file", file),
					zap.String("repo", bundle.RepoID),
					zap.String("bundleID", bundle.BundleID),
					zap.Error(err),
				)
				continue
			}
			select {
			case chans.error <- errorHit{
				error: err,
				file:  file,
			}:
			case <-chans.done:
			}
		}
		concurrencyControl <- struct{}{}
		go uploadBundleFile(ctx, file, cafsArchive, fileReader, chans)
	}
	/* once the buffered channel semaphore is filled with sentinel entries,
	 * all `uploadBundleFile` goroutines have exited.
	 */
	for i := 0; i < cap(concurrencyControl); i++ {
		concurrencyControl <- struct{}{}
	}
	chans.doneOk <- struct{}{}
}

func uploadBundle(ctx context.Context, bundle *Bundle, bundleEntriesPerFile uint, getKeys func() ([]string, error)) error {
	// Walk the entire tree
	// TODO: #53 handle large file count
	if getKeys == nil {
		getKeys = func() ([]string, error) {
			return bundle.ConsumableStore.Keys(context.Background())
		}
	}
	files, err := getKeys()
	if err != nil {
		return err
	}
	cafsArchive, err := cafs.New(
		cafs.LeafSize(bundle.BundleDescriptor.LeafSize),
		cafs.Backend(bundle.BlobStore),
		cafs.ConcurrentFlushes(bundle.concurrentFileUploads/fileUploadsPerFlush),
	)
	if err != nil {
		return err
	}

	// Upload the files and the bundle list
	err = bundle.InitializeBundleID()
	if err != nil {
		return err
	}

	filePackedC := make(chan filePacked)
	errorC := make(chan errorHit)
	doneC := make(chan struct{})
	doneOkC := make(chan struct{})
	defer close(doneC)

	go uploadBundleFiles(ctx, bundle, files, cafsArchive, uploadBundleChans{
		filePacked: filePackedC,
		error:      errorC,
		done:       doneC,
		doneOk:     doneOkC,
	})

	if MemProfDir != "" {
		var f *os.File
		path := filepath.Join(MemProfDir, "upload_bundle.mem.prof")
		f, err = os.Create(path)
		if err != nil {
			return err
		}
		err = pprof.Lookup("heap").WriteTo(f, 0)
		if err != nil {
			return err
		}
		f.Close()
	}
	filePackedList := make([]filePacked, 0, len(files))
	for {
		var gotDoneSignal bool
		select {
		case f := <-filePackedC:
			log.Printf("Uploaded file:%s, duplicate:%t, key:%s, keys:%d", f.name, f.duplicate, f.hash, len(f.keys))
			filePackedList = append(filePackedList, f)
		case e := <-errorC:
			fmt.Printf("Bundle upload failed. Failed to upload file %s err: %s", e.file, e.error)
			return e.error
		case <-doneOkC:
			gotDoneSignal = true
		}
		if gotDoneSignal {
			break
		}
	}
	fileList := make([]model.BundleEntry, 0, bundleEntriesPerFile)
	for packedFileIdx, packedFile := range filePackedList {
		fileList = append(fileList, filePacked2BundleEntry(packedFile))
		// Write the bundle entry file if reached max or the last one
		if packedFileIdx == len(filePackedList)-1 || len(fileList) == int(bundleEntriesPerFile) {
			err = uploadBundleEntriesFileList(ctx, bundle, fileList)
			if err != nil {
				bundle.l.Error("Bundle upload failed.  Failed to upload bundle entries list.",
					zap.Error(err),
				)
				return err
			}
			fileList = fileList[:0]
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
