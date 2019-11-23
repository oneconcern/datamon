/*
 * Copyright © 2019 One Concern
 *
 */

package core

import (
	"github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/storage"
)

// GetRepoStore extracts the metadata store from some context's stores
//
// NOTE: this is redundant with GetBundleStore
func GetRepoStore(stores context.Stores) storage.Store {
	return getMetaStore(stores)
}
