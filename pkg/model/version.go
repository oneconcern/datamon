/*
 * Copyright © 2019 One Concern
 *
 */

package model

const (
	// CurrentBundleVersion indicates the version of the bundle model
	//
	// Note that version numbering is an integer, not a semver string.
	//
	// TODO(fred): we should move to semver and use some semver comparison package whenever we want to compare versions.
	//
	// Change log from version 1:
	// - added support for diamond workflow (non breaking)
	CurrentBundleVersion uint64 = 2
)
