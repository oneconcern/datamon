// Copyright © 2018 One Concern

package core

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"

	//	"regexp"

	"github.com/oneconcern/datamon/pkg/cafs"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

const (
	fileDownloadsPerConcurrentChunks = 3
)

func getBundleIDFromPath(path string) (string, error) {
	info, err := model.GetConsumableStorePathMetadata(path)
	if err != nil {
		return "", err
	}
	if info.Type != model.ConsumableStorePathTypeDescriptor {
		return "", nil
	}
	return info.BundleID, nil
}

func setBundleIDFromConsumableStore(ctx context.Context, bundle *Bundle) error {
	var bundleID string
	keys, err := bundle.ConsumableStore.Keys(ctx)
	if err != nil {
		return err
	}
	for _, key := range keys {
		bundleID, err = getBundleIDFromPath(key)
		if err != nil {
			return err
		}
		if bundleID != "" {
			bundle.BundleID = bundleID
			return nil
		}
	}
	return fmt.Errorf("didn't find bundle descriptor")
}

type consumableStoreMetadataKeysInfo struct {
	bundleID   string
	descriptor string
	filelists  []string
}

func getConsumableStoreMetadataKeysInfo(ctx context.Context, bundle *Bundle) (consumableStoreMetadataKeysInfo, error) {
	// todo: use KeysPrefix() after that's implemented.
	keys, err := bundle.ConsumableStore.Keys(ctx)
	if err != nil {
		return consumableStoreMetadataKeysInfo{}, err
	}
	var bundleID string
	var descriptor string
	filelists := make([]string, 0)
	for _, key := range keys {

		info, err := model.GetConsumableStorePathMetadata(key)
		if err != nil {
			_, ok := err.(model.ConsumableStorePathMetadataErr)
			if ok {
				continue
			}
			return consumableStoreMetadataKeysInfo{}, err
		}
		switch info.Type {
		case model.ConsumableStorePathTypeDescriptor:
			if bundleID != "" {
				return consumableStoreMetadataKeysInfo{}, fmt.Errorf("expected at most one descriptor")
			}
			bundleID = info.BundleID
			descriptor = key
		case model.ConsumableStorePathTypeFileList:
			filelists = append(filelists, key)
		default:
			return consumableStoreMetadataKeysInfo{}, fmt.Errorf("unexpected consumable store metadata path type")
		}
	}
	if bundleID == "" {
		return consumableStoreMetadataKeysInfo{}, fmt.Errorf("bundle id not found")
	}
	return consumableStoreMetadataKeysInfo{
		bundleID:   bundleID,
		descriptor: descriptor,
		filelists:  filelists,
	}, nil
}

func unpackBundleDescriptor(ctx context.Context, bundle *Bundle, publish bool) error {
	var err error
	var bundleDescriptorBuffer []byte
	var rdr io.Reader
	switch {
	case publish:
		if bundle.MetaStore == nil || bundle.ConsumableStore == nil {
			return fmt.Errorf("can't publish without both meta and consumable stores")
		}

		bundleDescriptorBuffer, err = storage.ReadTee(ctx,
			bundle.MetaStore, model.GetArchivePathToBundle(bundle.RepoID, bundle.BundleID),
			bundle.ConsumableStore, model.GetConsumablePathToBundle(bundle.BundleID))
		if err != nil {
			return err
		}

	case bundle.MetaStore != nil:
		rdr, err = bundle.MetaStore.Get(ctx, model.GetArchivePathToBundle(bundle.RepoID, bundle.BundleID))
		if err != nil {
			return err
		}
		bundleDescriptorBuffer, err = ioutil.ReadAll(rdr)
		if err != nil {
			return err
		}
	default:
		rdr, err = bundle.ConsumableStore.Get(ctx, model.GetConsumablePathToBundle(bundle.BundleID))
		if err != nil {
			return err
		}
		bundleDescriptorBuffer, err = ioutil.ReadAll(rdr)
		if err != nil {
			return err
		}
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
	publish bool,
	i uint64,
) {
	var bundleEntries model.BundleEntries
	var err error
	var bundleEntriesBuffer []byte
	var rdr io.Reader

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

	archivePathToBundleFileList := model.GetArchivePathToBundleFileList(bundle.RepoID, bundle.BundleID, i)
	consumablePathToBundleFileList := model.GetConsumablePathToBundleFileList(bundle.BundleID, i)

	switch {
	case publish:
		if bundle.MetaStore == nil || bundle.ConsumableStore == nil {
			sendErr(fmt.Errorf("can't publish without both meta and consumable stores"))
			return
		}
		bundleEntriesBuffer, err = storage.ReadTee(ctx,
			bundle.MetaStore, archivePathToBundleFileList,
			bundle.ConsumableStore, consumablePathToBundleFileList)
		if err != nil {
			sendErr(err)
			return
		}
	case bundle.MetaStore != nil:
		rdr, err = bundle.MetaStore.Get(ctx, archivePathToBundleFileList)
		if err != nil {
			sendErr(err)
			return
		}
		bundleEntriesBuffer, err = ioutil.ReadAll(rdr)
		if err != nil {
			sendErr(err)
			return
		}
	default:
		rdr, err = bundle.ConsumableStore.Get(ctx, consumablePathToBundleFileList)
		if err != nil {
			sendErr(err)
			return
		}
		bundleEntriesBuffer, err = ioutil.ReadAll(rdr)
		if err != nil {
			sendErr(err)
			return
		}
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
	publish bool,
) {
	var i uint64
	concurrencyControl := make(chan struct{}, bundle.concurrentFilelistDownloads)
	chans.concurrencyControl = concurrencyControl
	for i = 0; i < bundle.BundleDescriptor.BundleEntriesFileCount; i++ {
		concurrencyControl <- struct{}{}
		go downloadBundleFileListFile(ctx, bundle, chans, publish, i)
	}
	for i := 0; i < cap(concurrencyControl); i++ {
		concurrencyControl <- struct{}{}
	}
	chans.doneOk <- struct{}{}
}

func unpackBundleFileList(ctx context.Context, bundle *Bundle,
	publish bool,
	bundleEntriesPerFile uint,
) error {

	bundle.l.Info("kicking off filelist download",
		zap.Int("concurrent Filelist Downloads", bundle.concurrentFilelistDownloads),
	)

	bundleEntriesC := make(chan bundleEntriesRes)
	errorC := make(chan error)
	doneC := make(chan struct{})
	doneOkC := make(chan struct{})

	defer close(doneC)

	go downloadBundleFileList(ctx, bundle, downloadBundleFileListChans{
		bundleEntries: bundleEntriesC,
		error:         errorC,
		doneOk:        doneOkC,
	}, publish)

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

func downloadBundleEntrySyncMaybeOverwrite(ctx context.Context, bundleEntry model.BundleEntry,
	bundle *Bundle,
	fs cafs.Fs,
	overwrite bool) error {
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
	var putParameterOverwrite bool
	if overwrite {
		err = bundle.ConsumableStore.Delete(ctx, bundleEntry.NameWithPath)
		if err != nil {
			bundle.l.Error("Failed to overwrite bundle entry: Delete to store",
				zap.String("name", bundleEntry.NameWithPath),
				zap.Error(err))
			return err
		}
		putParameterOverwrite = storage.IfNotPresent
	} else {
		putParameterOverwrite = storage.IfNotPresent
	}
	err = bundle.ConsumableStore.Put(ctx, bundleEntry.NameWithPath, reader, putParameterOverwrite)
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

func downloadBundleEntrySync(ctx context.Context, bundleEntry model.BundleEntry,
	bundle *Bundle,
	fs cafs.Fs) error {
	return downloadBundleEntrySyncMaybeOverwrite(ctx, bundleEntry, bundle, fs, false)
}

func deleteBundleEntrySync(ctx context.Context, bundleEntry model.BundleEntry,
	bundle *Bundle) error {
	bundle.l.Info("starting bundle entry delete",
		zap.String("name", bundleEntry.NameWithPath))
	err := bundle.ConsumableStore.Delete(ctx, bundleEntry.NameWithPath)
	if err != nil {
		bundle.l.Error("Failed to deleted bundle entry: store Delete",
			zap.String("name", bundleEntry.NameWithPath),
			zap.Error(err))
		return err
	}
	bundle.l.Info("deleted bundle entry",
		zap.String("name", bundleEntry.NameWithPath))
	return nil
}

// dupe: deleteBundleEntry
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

// dupe: deleteBundleEntry
func downloadBundleEntryOverwrite(ctx context.Context, bundleEntry model.BundleEntry,
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
	err := downloadBundleEntrySyncMaybeOverwrite(ctx, bundleEntry, bundle, fs, true)
	if err != nil {
		reportError(err)
		return
	}
}

func deleteBundleEntry(ctx context.Context, bundleEntry model.BundleEntry,
	bundle *Bundle,
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
	err := deleteBundleEntrySync(ctx, bundleEntry, bundle)
	if err != nil {
		reportError(err)
		return
	}
}

func downloadBundleEntries(ctx context.Context, bundle *Bundle,
	selectionPredicate func(string) (bool, error),
	bundleDest *Bundle,
	fs cafs.Fs,
	chans downloadBundleChans) {
	var diff BundleDiff
	var selectionPredicateOk bool
	var err error
	reportError := func(err error) {
		chans.error <- errorHit{
			err,
			"",
		}
	}
	if bundleDest != nil {
		diff, err = diffBundles(bundleDest, bundle)
		if err != nil {
			reportError(err)
			return
		}
	}
	concurrentFileDownloads := bundle.concurrentFileDownloads
	if concurrentFileDownloads < 1 {
		concurrentFileDownloads = 1
	}
	concurrencyControl := make(chan struct{}, concurrentFileDownloads)
	chans.concurrencyControl = concurrencyControl
	if bundleDest == nil {
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
	} else {
		bundle.l.Info("downloading diff entries",
			zap.Int("num", len(diff.Entries)))
		for _, de := range diff.Entries {
			concurrencyControl <- struct{}{}
			switch de.Type {
			case DiffEntryTypeAdd:
				bundle.l.Info("adding added entry",
					zap.String("name Additional", de.Additional.NameWithPath),
					zap.String("name Existing", de.Existing.NameWithPath),
				)
				go downloadBundleEntry(ctx, de.Additional, bundleDest, fs, chans)
			case DiffEntryTypeDel:
				bundle.l.Info("deleting deleted entry",
					zap.String("name Additional", de.Additional.NameWithPath),
					zap.String("name Existing", de.Existing.NameWithPath),
				)
				go deleteBundleEntry(ctx, de.Existing, bundleDest, chans)
			case DiffEntryTypeDif:
				bundle.l.Info("updating diff entry",
					zap.String("name Additional", de.Additional.NameWithPath),
					zap.String("name Existing", de.Existing.NameWithPath),
				)
				go downloadBundleEntryOverwrite(ctx, de.Additional, bundleDest, fs, chans)
			default:
				reportError(fmt.Errorf("programming error: unknown diff entry type"))
				<-concurrencyControl
			}
		}
	}
	for i := 0; i < cap(concurrencyControl); i++ {
		concurrencyControl <- struct{}{}
	}
	chans.doneOk <- struct{}{}
}

func unpackDataFiles(ctx context.Context, bundle *Bundle,
	bundleDest *Bundle,
	selectionPredicate func(string) (bool, error)) error {
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
		bundleDest,
		fs,
		downloadBundleChans{
			error:  errC,
			doneOk: doneOkC,
		})
	select {
	case eh := <-errC:
		return eh.error
	case <-doneOkC:
	}

	// rewrite destination bundle metadata
	if bundleDest != nil {
		info, err := getConsumableStoreMetadataKeysInfo(ctx, bundleDest)
		if err != nil {
			return err
		}
		err = bundleDest.ConsumableStore.Delete(ctx, info.descriptor)
		if err != nil {
			return err
		}
		for _, filelist := range info.filelists {
			err = bundleDest.ConsumableStore.Delete(ctx, filelist)
			if err != nil {
				return err
			}
		}
		publishMetadataBundle := New(NewBDescriptor(),
			Repo(bundle.RepoID),
			MetaStore(bundle.MetaStore),
			ConsumableStore(bundleDest.ConsumableStore),
			BlobStore(bundle.BlobStore),
			BundleID(bundle.BundleID),
		)
		err = PublishMetadata(ctx, publishMetadataBundle)
		if err != nil {
			return err
		}
	} // bundleDest != nil

	return nil
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
