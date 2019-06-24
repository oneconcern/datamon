// Copyright © 2018 One Concern

package core

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/oneconcern/datamon/pkg/cafs"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
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

type downloadBundleChans struct {
	error              chan<- errorHit
	done               <-chan struct{}
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
		select {
		case chans.error <- errorHit{
			err,
			bundleEntry.NameWithPath,
		}:
		case <-chans.done:
		}
	}
	err := downloadBundleEntrySync(ctx, bundleEntry, bundle, fs)
	if err != nil {
		reportError(err)
		return
	}
}

func downloadBundleEntries(ctx context.Context, bundle *Bundle,
	fs cafs.Fs,
	chans downloadBundleChans) {
	concurrentFileDownloads := bundle.concurrentFileDownloads
	if concurrentFileDownloads < 1 {
		concurrentFileDownloads = 1
	}
	concurrencyControl := make(chan struct{}, concurrentFileDownloads)
	chans.concurrencyControl = concurrencyControl
	bundle.l.Info("downloading bundle entries",
		zap.Int("num", len(bundle.BundleEntries)))
	for _, b := range bundle.BundleEntries {
		concurrencyControl <- struct{}{}
		go downloadBundleEntry(ctx, b, bundle, fs, chans)
	}
	for i := 0; i < cap(concurrencyControl); i++ {
		concurrencyControl <- struct{}{}
	}
	chans.doneOk <- struct{}{}
}

func unpackDataFiles(ctx context.Context, bundle *Bundle) error {
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
	doneC := make(chan struct{})
	doneOkC := make(chan struct{})
	defer close(doneC)
	go downloadBundleEntries(
		ctx,
		bundle,
		fs,
		downloadBundleChans{
			error:  errC,
			done:   doneC,
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
