// Copyright © 2018 One Concern
package core

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/oneconcern/datamon/pkg/dlogger"

	"go.uber.org/zap"

	"github.com/oneconcern/datamon/pkg/cafs"
	"gopkg.in/yaml.v2"

	"github.com/segmentio/ksuid"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
)

type errString string

func (e errString) Error() string { return string(e) }

const (
	ErrNotFound errString = "not found"
)

var MemProfDir string

// ArchiveBundle represents the bundle in it's archive state
type Bundle struct {
	RepoID                  string
	BundleID                string
	MetaStore               storage.Store
	ConsumableStore         storage.Store
	BlobStore               storage.Store
	cafs                    cafs.Fs
	BundleDescriptor        model.BundleDescriptor
	BundleEntries           []model.BundleEntry
	Streamed                bool
	l                       *zap.Logger
	SkipOnError             bool // When uploading files
	concurrentFileUploads   int
	concurrentFileDownloads int
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

func MetaStore(store storage.Store) BundleOption {
	return func(b *Bundle) {
		b.MetaStore = store
	}
}
func ConsumableStore(store storage.Store) BundleOption {
	return func(b *Bundle) {
		b.ConsumableStore = store
	}
}
func BlobStore(store storage.Store) BundleOption {
	return func(b *Bundle) {
		b.BlobStore = store
	}
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

func New(bd *model.BundleDescriptor, bundleOps ...BundleOption) *Bundle {
	b := Bundle{
		RepoID:                  "",
		BundleID:                "",
		MetaStore:               nil,
		ConsumableStore:         nil,
		BlobStore:               nil,
		BundleDescriptor:        *bd,
		BundleEntries:           make([]model.BundleEntry, 0, 1024),
		Streamed:                false,
		concurrentFileUploads:   20,
		concurrentFileDownloads: 10,
	}

	b.l, _ = dlogger.GetLogger("info")

	for _, bApply := range bundleOps {
		bApply(&b)
	}
	if b.Streamed {
		ls := b.BundleDescriptor.LeafSize
		fs, _ := cafs.New(
			cafs.LeafSize(ls),
			cafs.LeafTruncation(b.BundleDescriptor.Version < 1),
			cafs.Backend(b.BlobStore),
		)
		b.cafs = fs
	}
	return &b
}

// Publish an bundle to a consumable store
func Publish(ctx context.Context, bundle *Bundle) error {
	err := PublishMetadata(ctx, bundle)
	if err != nil {
		return fmt.Errorf("failed to publish, err:%s", err)
	}
	if !bundle.Streamed {
		err = unpackDataFiles(ctx, bundle)
		if err != nil {
			return fmt.Errorf("failed to unpack data files, err:%s", err)
		}
	}
	return nil
}

// PublishMetadata from the archive to the consumable store
func PublishMetadata(ctx context.Context, bundle *Bundle) error {
	err := unpackBundleDescriptor(ctx, bundle)
	if err != nil {
		return err
	}

	err = unpackBundleFileList(ctx, bundle)
	if err != nil {
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
	err := RepoExists(bundle.RepoID, bundle.MetaStore)
	if err != nil {
		return err
	}
	return uploadBundle(ctx, bundle, bundleEntriesPerFile, getKeys)
}

func PopulateFiles(ctx context.Context, bundle *Bundle) error {
	e := RepoExists(bundle.RepoID, bundle.MetaStore)
	if e != nil {
		return e
	}
	reader, err := bundle.MetaStore.Get(ctx, model.GetArchivePathToBundle(bundle.RepoID, bundle.BundleID))
	if err != nil {
		fmt.Printf("Failed to download the bundle descriptor: %s", err)
		return err
	}
	defer reader.Close()
	object, err := ioutil.ReadAll(reader)
	if err != nil {
		fmt.Printf("Failed to read the bundle descriptor: %s", err)
		return err
	}
	// Unmarshal the file
	err = yaml.Unmarshal(object, &bundle.BundleDescriptor)
	if err != nil {
		fmt.Printf("Failed to unmarshal the bundle descriptor: %s", err)
		return err
	}

	// Download the files json
	var i uint64
	for i = 0; i < bundle.BundleDescriptor.BundleEntriesFileCount; i++ {
		r, err := bundle.MetaStore.Get(ctx, model.GetArchivePathToBundleFileList(bundle.RepoID, bundle.BundleID, i))
		if err != nil {
			fmt.Printf("Failed to download the bundle files: %s", err)
			return err
		}
		object, err = ioutil.ReadAll(r)
		if err != nil {
			fmt.Printf("Failed to read the bundle files: %s", err)
			return err
		}
		var bundleEntries model.BundleEntries
		err = yaml.Unmarshal(object, &bundleEntries)
		if err != nil {
			return err
		}
		bundle.BundleEntries = append(bundle.BundleEntries, bundleEntries.BundleEntries...)
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
	return b.MetaStore.Has(ctx, model.GetArchivePathToBundle(b.RepoID, b.BundleID))
}
