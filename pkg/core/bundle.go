// Copyright © 2018 One Concern
package core

import (
	"context"
	"time"

	"github.com/oneconcern/datamon/pkg/cafs"

	"github.com/segmentio/ksuid"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
)

// ArchiveBundle represents the bundle in it's archive state
type Bundle struct {
	RepoID           string
	BundleID         string
	MetaStore        storage.Store
	ConsumableStore  storage.Store
	BlobStore        storage.Store
	BundleDescriptor model.BundleDescriptor
	BundleEntries    []model.BundleEntry
}

// SetBundleID for the bundle
func (bundle *Bundle) setBundleID(id string) {
	bundle.BundleID = id
	bundle.BundleDescriptor.ID = id
}

// InitializeBundleID create and set a new bundle ID
func (bundle *Bundle) InitializeBundleID() error {
	id, err := ksuid.NewRandom()
	if err != nil {
		return err
	}
	bundle.setBundleID(id.String())
	return nil
}

func (bundle *Bundle) GetBundleEntries() []model.BundleEntry {
	return bundle.BundleEntries
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

func New(bd *model.BundleDescriptor, bundleOps ...BundleOption) *Bundle {
	b := Bundle{
		RepoID:           "",
		BundleID:         "",
		MetaStore:        nil,
		ConsumableStore:  nil,
		BlobStore:        nil,
		BundleDescriptor: *bd,
		BundleEntries:    make([]model.BundleEntry, 0, 1024),
	}
	for _, bApply := range bundleOps {
		bApply(&b)
	}
	return &b
}

// Publish an bundle to a consumable store
func Publish(ctx context.Context, bundle *Bundle) error {
	err := PublishMetadata(ctx, bundle)
	if err != nil {
		return err
	}

	err = unpackDataFiles(ctx, bundle)
	if err != nil {
		return err
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
	err := RepoExists(bundle.RepoID, bundle.MetaStore)
	if err != nil {
		return err
	}
	return uploadBundle(ctx, bundle)
}
