package core

import (
	context2 "github.com/oneconcern/datamon/pkg/context"
)

// PurgeBuildReverseIndex creates or update a reverse-lookip index
// of all used blob keys.
func PurgeBuildReverseIndex(stores context2.Stores, opts ...PurgeOption) error {
	return nil // TODO
}

// PurgeUnlock removes the purge job lock from the metadata store.
func PurgeUnlock(stores context2.Stores) error {
	return nil // TODO
}

// PurgeLock sets a purge job lock on the metadata store.
func PurgeLock(stores context2.Stores, opts ...PurgeOption) error {
	return nil // TODO
}

// PurgeDeleteUnused deletes blob entries that are not referenced by the reserve-lookup index.
func PurgeDeleteUnused(stores context2.Stores, opts ...PurgeOption) error {
	return nil // TODO
}
