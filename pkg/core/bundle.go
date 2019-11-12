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

var MemProfDir string

// Bundle represents the bundle in its archived state
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
}

// SetBundleID for the bundle
func (b *Bundle) setBundleID(id string) {
	b.BundleID = id
	b.BundleDescriptor.ID = id
}

// InitializeBundleID create and set a new bundle ID
func (b *Bundle) InitializeBundleID() error {
	id, err := ksuid.NewRandom()
	if err != nil {
		return err
	}
	b.setBundleID(id.String())
	return nil
}

func (b *Bundle) GetBundleEntries() []model.BundleEntry {
	return b.BundleEntries
}

type BundleOption func(*Bundle)

type BundleDescriptorOption func(descriptor *model.BundleDescriptor)

func Message(m string) BundleDescriptorOption {
	return func(b *model.BundleDescriptor) {
		b.Message = m
	}
}

func Contributors(c []model.Contributor) BundleDescriptorOption {
	return func(b *model.BundleDescriptor) {
		b.Contributors = c
	}
}

func Contributor(c model.Contributor) BundleDescriptorOption {
	return Contributors([]model.Contributor{c})
}

func Parents(p []string) BundleDescriptorOption {
	return func(b *model.BundleDescriptor) {
		b.Parents = p
	}
}
func Deduplication(d string) BundleDescriptorOption {
	return func(b *model.BundleDescriptor) {
		b.Deduplication = d
	}
}

func NewBDescriptor(descriptorOps ...BundleDescriptorOption) *model.BundleDescriptor {
	bd := model.BundleDescriptor{
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
	for _, apply := range descriptorOps {
		apply(&bd)
	}
	return &bd
}

func Repo(r string) BundleOption {
	return func(b *Bundle) {
		b.RepoID = r
	}
}

func ConsumableStore(store storage.Store) BundleOption {
	return func(b *Bundle) {
		b.ConsumableStore = store
	}
}
func ContextStores(cs context2.Stores) BundleOption {
	return func(b *Bundle) {
		b.contextStores = cs
	}
}
func (b *Bundle) BlobStore() storage.Store {
	return getBlobStore(b.contextStores)
}

func getBlobStore(stores context2.Stores) storage.Store {
	return stores.Blob()
}
func (b *Bundle) MetaStore() storage.Store {
	return getMetaStore(b.contextStores)
}
func getMetaStore(stores context2.Stores) storage.Store {
	return stores.Metadata()
}
func (b *Bundle) VMetaStore() storage.Store {
	return getVMetaStore(b.contextStores)
}
func getVMetaStore(stores context2.Stores) storage.Store {
	return stores.VMetadata()
}
func (b *Bundle) WALStore() storage.Store {
	return getWALStore(b.contextStores)
}
func getWALStore(stores context2.Stores) storage.Store {
	return stores.Wal()
}
func (b *Bundle) ReadLogStore() storage.Store {
	return getReadLogStore(b.contextStores)
}
func getReadLogStore(stores context2.Stores) storage.Store {
	return stores.ReadLog()
}

func BundleID(bID string) BundleOption {
	return func(b *Bundle) {
		b.BundleID = bID
	}
}

func Streaming(s bool) BundleOption {
	return func(b *Bundle) {
		b.Streamed = s
	}
}

func SkipMissing(s bool) BundleOption {
	return func(b *Bundle) {
		b.SkipOnError = s
	}
}

func Logger(l *zap.Logger) BundleOption {
	return func(b *Bundle) {
		b.l = l
	}
}

func ConcurrentFileUploads(concurrentFileUploads int) BundleOption {
	return func(b *Bundle) {
		b.concurrentFileUploads = concurrentFileUploads
	}
}

func ConcurrentFileDownloads(concurrentFileDownloads int) BundleOption {
	return func(b *Bundle) {
		b.concurrentFileDownloads = concurrentFileDownloads
	}
}

func ConcurrentFilelistDownloads(concurrentFilelistDownloads int) BundleOption {
	return func(b *Bundle) {
		b.concurrentFilelistDownloads = concurrentFilelistDownloads
	}
}

func defaultBundle() Bundle {
	return Bundle{
		RepoID:                      "",
		BundleID:                    "",
		ConsumableStore:             nil,
		BundleEntries:               make([]model.BundleEntry, 0, 1024),
		Streamed:                    false,
		concurrentFileUploads:       20,
		concurrentFileDownloads:     10,
		concurrentFilelistDownloads: 10,
	}
}

func NewBundle(bd *model.BundleDescriptor, bundleOps ...BundleOption) *Bundle {
	b := defaultBundle()
	b.BundleDescriptor = *bd

	b.l, _ = dlogger.GetLogger("info")

	for _, bApply := range bundleOps {
		bApply(&b)
	}
	if b.Streamed {
		ls := b.BundleDescriptor.LeafSize
		fs, _ := cafs.New(
			cafs.LeafSize(ls),
			cafs.LeafTruncation(b.BundleDescriptor.Version < 1),
			cafs.Backend(b.BlobStore()),
		)
		b.cafs = fs
	}
	return &b
}

// Publish an bundle to a consumable store
func Publish(ctx context.Context, bundle *Bundle) error {
	return implPublish(ctx, bundle, defaultBundleEntriesPerFile, func(s string) (bool, error) { return true, nil })
}

func PublishSelectBundleEntries(ctx context.Context, bundle *Bundle, selectionPredicate func(string) (bool, error)) error {
	return implPublish(ctx, bundle, defaultBundleEntriesPerFile, func(s string) (bool, error) { return true, nil })
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
func Upload(ctx context.Context, bundle *Bundle) error {
	return implUpload(ctx, bundle, defaultBundleEntriesPerFile, nil)
}

// Upload specified keys (files) within a bundle's consumable store
func UploadSpecificKeys(ctx context.Context, bundle *Bundle, getKeys func() ([]string, error)) error {
	return implUpload(ctx, bundle, defaultBundleEntriesPerFile, getKeys)
}

// implementation of Upload() with some additional parameters for test
func implUpload(ctx context.Context, bundle *Bundle, bundleEntriesPerFile uint, getKeys func() ([]string, error)) error {
	err := RepoExists(bundle.RepoID, bundle.contextStores)
	if err != nil {
		return err
	}
	return uploadBundle(ctx, bundle, bundleEntriesPerFile, getKeys)
}

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

func (b *Bundle) Exists(ctx context.Context) (bool, error) {
	return b.MetaStore().Has(ctx, model.GetArchivePathToBundle(b.RepoID, b.BundleID))
}

func Diff(ctx context.Context, bundleExisting *Bundle, bundleAdditional *Bundle) (BundleDiff, error) {
	if err := PopulateFiles(ctx, bundleExisting); err != nil {
		return BundleDiff{}, err
	}
	if err := PopulateFiles(ctx, bundleAdditional); err != nil {
		return BundleDiff{}, err
	}
	return diffBundles(bundleExisting, bundleAdditional)
}

func Update(ctx context.Context, bundleSrc *Bundle, bundleDest *Bundle) error {
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

func GetBundleStore(stores context2.Stores) storage.Store {
	return getMetaStore(stores)
}
