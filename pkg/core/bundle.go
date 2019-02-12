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
	ArchiveStore     storage.Store
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

// NewArchiveBundle returns a new archive bundle
func NewBundle(repo string, bundle string, archiveStore storage.Store, consumableStore storage.Store, blobStore storage.Store) *Bundle {
	return &Bundle{
		BundleDescriptor: model.BundleDescriptor{
			LeafSize:               cafs.DefaultLeafSize,
			ID:                     "",
			Message:                "",
			Parents:                nil,
			Timestamp:              time.Now(),
			Contributors:           nil,
			BundleEntriesFileCount: 0,
		},
		RepoID:          repo,
		BundleID:        bundle,
		ArchiveStore:    archiveStore,
		ConsumableStore: consumableStore,
		BlobStore:       blobStore,
	}
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
	return uploadBundle(ctx, bundle)
}
