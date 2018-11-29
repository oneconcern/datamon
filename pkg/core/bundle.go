// Copyright © 2018 One Concern
package core

import (
	"context"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
)

// Represents the bundle in it's archive state
type ArchiveBundle struct {
	repoID           string
	bundleID         string
	store            storage.Store
	bundleDescriptor model.Bundle
	bundleEntries    []model.BundleEntry
}

type ConsumableBundle struct {
	Store storage.Store
}

func Publish(ctx context.Context, archiveBundle *ArchiveBundle, consumableBundle ConsumableBundle) error {
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

func NewArchiveBundle(repo string, bundle string, store storage.Store) (*ArchiveBundle, error) {
	return &ArchiveBundle{
		repoID:   repo,
		bundleID: bundle,
		store:    store,
	}, nil
}
