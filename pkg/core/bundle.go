// Copyright © 2018 One Concern

package core

import (
	"context"
	"fmt"
	"time"

	context2 "github.com/oneconcern/datamon/pkg/context"

	"github.com/oneconcern/datamon/pkg/dlogger"

	"go.uber.org/zap"

	"github.com/oneconcern/datamon/pkg/cafs"

	"github.com/segmentio/ksuid"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
)

// Bundle represents a bundle in its archived state.
//
// A bundle is a point in time read-only view of a rep:branch and is composed
// of individual files. Analogous to a commit in git.
type Bundle struct {
	RepoID                      string
	BundleID                    string
	ConsumableStore             storage.Store
	contextStores               context2.Stores
	cafs                        cafs.Fs
	BundleDescriptor            model.BundleDescriptor
	BundleEntries               []model.BundleEntry
	Streamed                    bool
	l                           *zap.Logger
	SkipOnError                 bool // When uploading files
	concurrentFileUploads       int
	concurrentFileDownloads     int
	concurrentFilelistDownloads int
	lruSize                     int // when Streamed = true
	withPrefetch                int // when Streamed = true
	withVerifyHash              bool
}

// SetBundleID for the bundle
func (b *Bundle) setBundleID(id string) {
	b.BundleID = id
	b.BundleDescriptor.ID = id
}

// InitializeBundleID creates and sets a new bundle ID
func (b *Bundle) InitializeBundleID() error {
	id, err := ksuid.NewRandom()
	if err != nil {
		return err
	}
	b.setBundleID(id.String())
	return nil
}

// GetBundleEntries retrieves all entries in a bundle
func (b *Bundle) GetBundleEntries() []model.BundleEntry {
	return b.BundleEntries
}

func defaultBundleDescriptor() *model.BundleDescriptor {
	return &model.BundleDescriptor{
		LeafSize:               cafs.DefaultLeafSize, // For now, fixed leaf size
		ID:                     "",
		Message:                "",
		Parents:                nil,
		Timestamp:              time.Now(),
		Contributors:           nil,
		BundleEntriesFileCount: 0,
		Version:                model.CurrentBundleVersion,
		Deduplication:          cafs.DeduplicationBlake,
	}
}

// NewBDescriptor builds a new default bundle descriptor
func NewBDescriptor(descriptorOps ...BundleDescriptorOption) *model.BundleDescriptor {
	bd := defaultBundleDescriptor()
	for _, apply := range descriptorOps {
		apply(bd)
	}
	return bd
}

// BlobStore defines the blob storage (part of the context) for a bundle
func (b *Bundle) BlobStore() storage.Store {
	return getBlobStore(b.contextStores)
}

func getBlobStore(stores context2.Stores) storage.Store {
	if stores == nil {
		return nil
	}
	return stores.Blob()
}

// MetaStore yields the metadata store for the current bundle context
func (b *Bundle) MetaStore() storage.Store {
	return getMetaStore(b.contextStores)
}

func getMetaStore(stores context2.Stores) storage.Store {
	if stores == nil {
		return nil
	}
	return stores.Metadata()
}

// VMetaStore yields the metadata store for the current bundle context
func (b *Bundle) VMetaStore() storage.Store {
	return getVMetaStore(b.contextStores)
}
func getVMetaStore(stores context2.Stores) storage.Store {
	if stores == nil {
		return nil
	}
	return stores.VMetadata()
}

// WALStore yields the Write Ahead Log storage for the bundle context
func (b *Bundle) WALStore() storage.Store {
	return getWALStore(b.contextStores)
}

func getWALStore(stores context2.Stores) storage.Store {
	if stores == nil {
		return nil
	}
	return stores.Wal()
}

// ReadLogStore yields the Read Log storage for the bundle context
func (b *Bundle) ReadLogStore() storage.Store {
	return getReadLogStore(b.contextStores)
}
func getReadLogStore(stores context2.Stores) storage.Store {
	if stores == nil {
		return nil
	}
	return stores.ReadLog()
}

func defaultBundle() Bundle {
	l, _ := dlogger.GetLogger("info")
	return Bundle{
		RepoID:                      "",
		BundleID:                    "",
		ConsumableStore:             nil,
		BundleEntries:               make([]model.BundleEntry, 0, 1024),
		Streamed:                    false,
		concurrentFileUploads:       20,
		concurrentFileDownloads:     10,
		concurrentFilelistDownloads: 10,
		l:                           l,

		// default cafs options
		withPrefetch:   1,
		withVerifyHash: true,
	}
}

// NewBundle creates a new bundle
func NewBundle(bd *model.BundleDescriptor, bundleOps ...BundleOption) *Bundle {
	b := defaultBundle()
	b.BundleDescriptor = *bd

	for _, bApply := range bundleOps {
		bApply(&b)
	}
	if b.Streamed {
		// prepare the content-addressable backend for this bundle
		fs, _ := cafs.New(
			cafs.LeafSize(b.BundleDescriptor.LeafSize),
			cafs.LeafTruncation(b.BundleDescriptor.Version < 1),
			cafs.Backend(b.BlobStore()),
			cafs.Logger(b.l),
			cafs.CacheSize(b.lruSize),
			cafs.Prefetch(b.withPrefetch),
			cafs.VerifyHash(b.withVerifyHash),
		)
		b.cafs = fs
	}
	return &b
}

// Publish a bundle to a consumable store
func Publish(ctx context.Context, bundle *Bundle) error {
	return implPublish(ctx, bundle, defaultBundleEntriesPerFile, func(s string) (bool, error) { return true, nil })
}

// PublishSelectBundleEntries publish a selected list of entries from a bundle to a ConsumableStore, based on a predicate filter
func PublishSelectBundleEntries(ctx context.Context, bundle *Bundle, selectionPredicate func(string) (bool, error)) error {
	return implPublish(ctx, bundle, defaultBundleEntriesPerFile, selectionPredicate)
}

// implementation of Publish() with some additional parameters for test
func implPublish(ctx context.Context, bundle *Bundle, bundleEntriesPerFile uint,
	selectionPredicate func(string) (bool, error)) error {
	err := implPublishMetadata(ctx, bundle, true, bundleEntriesPerFile)
	if err != nil {
		return fmt.Errorf("failed to publish, err:%s", err)
	}
	if !bundle.Streamed {
		err = unpackDataFiles(ctx, bundle, nil, selectionPredicate)
		if err != nil {
			return fmt.Errorf("failed to unpack data files, err:%s", err)
		}
	}
	return nil
}

// PublishMetadata from the archive to the consumable store
func PublishMetadata(ctx context.Context, bundle *Bundle) error {
	return implPublishMetadata(ctx, bundle, true, defaultBundleEntriesPerFile)
}

// DownloadMetadata from the archive to main memory
func DownloadMetadata(ctx context.Context, bundle *Bundle) error {
	return implPublishMetadata(ctx, bundle, false, defaultBundleEntriesPerFile)
}

// implementation of PublishMetadata() with some additional parameters for test
func implPublishMetadata(ctx context.Context, bundle *Bundle,
	publish bool,
	bundleEntriesPerFile uint,
) error {
	if bundle.BundleID == "" {
		if bundle.ConsumableStore != nil {
			if err := setBundleIDFromConsumableStore(ctx, bundle); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("no bundle id set and consumable store not present")
		}
	}
	if err := unpackBundleDescriptor(ctx, bundle, publish); err != nil {
		return err
	}
	// async needs bundleEntriesPerFile
	if err := unpackBundleFileList(ctx, bundle, publish, bundleEntriesPerFile); err != nil {
		return err
	}
	return nil
}

// Upload an bundle to archive
func Upload(ctx context.Context, bundle *Bundle, opts ...Option) error {
	return implUpload(ctx, bundle, defaultBundleEntriesPerFile, nil, opts...)
}

// UploadSpecificKeys uploads some specified keys (files) within a bundle's consumable store
func UploadSpecificKeys(ctx context.Context, bundle *Bundle, getKeys func() ([]string, error), opts ...Option) error {
	return implUpload(ctx, bundle, defaultBundleEntriesPerFile, getKeys, opts...)
}

// implementation of Upload() with some additional parameters for test
func implUpload(ctx context.Context, bundle *Bundle, bundleEntriesPerFile uint, getKeys func() ([]string, error), opts ...Option) error {
	if err := RepoExists(bundle.RepoID, bundle.contextStores); err != nil {
		return err
	}
	return uploadBundle(ctx, bundle, bundleEntriesPerFile, getKeys, opts...)
}

// PopulateFiles populates a ConsumableStore with a bundle's files
func PopulateFiles(ctx context.Context, bundle *Bundle) error {
	switch {
	case bundle.ConsumableStore != nil && bundle.MetaStore() != nil:
		return fmt.Errorf("ambiguous bundle to populate files:  consumable store and meta store both exist")
	case bundle.ConsumableStore != nil:
		if bundle.BundleID == "" {
			if err := setBundleIDFromConsumableStore(ctx, bundle); err != nil {
				return err
			}
		}
	case bundle.MetaStore() != nil:
		if err := RepoExists(bundle.RepoID, bundle.contextStores); err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid bundle to populate files:  neither consumable store nor meta store exists")
	}
	if err := implPublishMetadata(ctx, bundle, false, defaultBundleEntriesPerFile); err != nil {
		return err
	}
	return nil
}

// PublishFile publish a single bundle file to a ConsumableStore
func PublishFile(ctx context.Context, bundle *Bundle, file string) error {
	err := PublishMetadata(ctx, bundle)
	if err != nil {
		return err
	}

	err = unpackDataFile(ctx, bundle, file)
	if err != nil {
		return err
	}
	return nil
}

// Exists checks for the existence of this bundle in the repository
func (b *Bundle) Exists(ctx context.Context) (bool, error) {
	return b.MetaStore().Has(ctx, model.GetArchivePathToBundle(b.RepoID, b.BundleID))
}

// Diff shows the differences between two bundles
func Diff(ctx context.Context, bundleExisting, bundleAdditional *Bundle) (BundleDiff, error) {
	if err := PopulateFiles(ctx, bundleExisting); err != nil {
		return BundleDiff{}, err
	}
	if err := PopulateFiles(ctx, bundleAdditional); err != nil {
		return BundleDiff{}, err
	}
	return diffBundles(bundleExisting, bundleAdditional)
}

// Update a destination bundle from a source bundle
func Update(ctx context.Context, bundleSrc, bundleDest *Bundle) error {
	if err := implPublishMetadata(ctx, bundleSrc, false, defaultBundleEntriesPerFile); err != nil {
		return err
	}
	if err := implPublishMetadata(ctx, bundleDest, false, defaultBundleEntriesPerFile); err != nil {
		return err
	}
	if err := unpackDataFiles(ctx, bundleSrc, bundleDest, nil); err != nil {
		return err
	}
	return nil
}

// GetBundleStore extracts the metadata store for bundles from some context's stores
func GetBundleStore(stores context2.Stores) storage.Store {
	return getMetaStore(stores)
}
