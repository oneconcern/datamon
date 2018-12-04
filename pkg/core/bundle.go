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
	repoID           string
	bundleID         string
	store            storage.Store
	bundleDescriptor model.Bundle
	bundleEntries    []model.BundleEntry
}

// SetBundleID for the bundle
func (bundle *Bundle) SetBundleID(id string) {
	bundle.bundleID = id
	bundle.bundleDescriptor.ID = id
}

// InitializeBundleID create and set a new bundle ID
func (bundle *Bundle) InitializeBundleID() error {
	id, err := ksuid.NewRandom()
	if err != nil {
		return err
	}
	bundle.SetBundleID(id.String())
	return nil
}

// NewArchiveBundle returns a new archive bundle
func NewBundle(repo string, bundle string, store storage.Store) *Bundle {
	return &Bundle{
		bundleDescriptor: model.Bundle{
			LeafSize:               cafs.DefaultLeafSize,
			ID:                     "",
			Message:                "",
			Parents:                nil,
			Timestamp:              time.Now(),
			Contributors:           nil,
			BundleEntriesFileCount: 0,
		},
		repoID:   repo,
		bundleID: bundle,
		store:    store,
	}
}

// Publish an archived bundle to a consumable bundle
func Publish(ctx context.Context, archiveBundle *Bundle, consumableBundle *Bundle) error {
	err := unpackBundleDescriptor(ctx, archiveBundle, consumableBundle)
	if err != nil {
		return err
	}

	err = unpackBundleFileList(ctx, archiveBundle, consumableBundle)
	if err != nil {
		return err
	}

	err = unpackDataFiles(ctx, archiveBundle, consumableBundle)
	if err != nil {
		return err
	}
	return nil
}

// Upload an archive bundle that is stored as a consumable bundle
func Upload(ctx context.Context, consumableBundle Bundle, archiveBundle *Bundle) error {
	return uploadBundle(ctx, consumableBundle, archiveBundle)
}
