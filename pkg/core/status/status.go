// Package status exports errors produced by the core package.
package status

import (
	"github.com/oneconcern/datamon/pkg/errors"
)

var (
	// ErrInterrupted signals that the current background processing has been interrupted
	ErrInterrupted = errors.New("background processing interrupted")

	// ErrNotFound indicates an object was not found
	ErrNotFound = errors.New("not found")

	// ErrUnexpectedUpdate indicates an update operation was attempted on some immutable store
	ErrUnexpectedUpdate = errors.New("unexpected update")

	// ErrConfigContext indicates an error while attempting to retrieve contexts from a remote config store
	ErrConfigContext = errors.New("error retrieving contexts from config store")

	// ErrPublish indicates an error while publishing (downloading) the set of files in the bundle
	ErrPublish = errors.New("failed to unpack data files")

	// ErrPublishMetadata indicates an error while publishing (downloading) the metadate for the bundle
	ErrPublishMetadata = errors.New("failed to publish metadata")

	// ErrNoBundleIDWithConsumable indicates some inconsistent bundle settings with both no bundleID and some ConsumableStore defined
	ErrNoBundleIDWithConsumable = errors.New("no bundle id set and consumable store not present")

	// ErrInvalidKsuid indicates that the bundleID used is not vallid and should parse as a ksuid.
	//
	// This may only appear when the feature to force (preserve) ksuid on bundle uploads is enabled.
	ErrInvalidKsuid = errors.New("invalid bundleID (ksuid) specified")

	// ErrAmbiguousBundle reports about some inconsistent settings with both consumable and metata store exist
	// when populating metadata.
	ErrAmbiguousBundle = errors.New("ambiguous bundle to populate files: consumable store and meta store both exist")

	// ErrInvalidBundle reports about some inconsistent settings with neither consumable nor metata store exist
	// when populating metadata.
	ErrInvalidBundle = errors.New("invalid bundle to populate files: neither consumable store nor meta store exists")

	// ErrBundleIDExists reports about a prohibited action to override an already existing bundleID on a given store.
	//
	// This may only appear when the feature to force (preserve) ksuid on bundle uploads is enabled.
	ErrBundleIDExists = errors.New("bundleID already exists on this store")
)
