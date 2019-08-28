// Copyright © 2018 One Concern

package core

import (
	"github.com/oneconcern/datamon/pkg/model"
)

const (
	DiffEntryTypeAdd = iota
	DiffEntryTypeDel
	DiffEntryTypeDif
)

type DiffEntryType uint

func (det DiffEntryType) String() string {
	diffEntryStrings := map[DiffEntryType]string{
		DiffEntryTypeAdd: "A",
		DiffEntryTypeDel: "D",
		DiffEntryTypeDif: "U",
	}
	return diffEntryStrings[det]
}

type DiffEntry struct {
	Type DiffEntryType
	// could use a method rather than storing Name in order to curb memory use
	Name       string
	Existing   model.BundleEntry
	Additional model.BundleEntry
}

type BundleDiff struct {
	Entries []DiffEntry
}

func diffBundles(bundleExisting *Bundle, bundleAdditional *Bundle) (BundleDiff, error) {
	diffEntries := make([]DiffEntry, 0)
	bundleEntriesExisting := make(map[string]model.BundleEntry, len(bundleExisting.BundleEntries))
	for _, bundleEntry := range bundleExisting.BundleEntries {
		bundleEntriesExisting[bundleEntry.NameWithPath] = bundleEntry
	}
	bundleEntriesAdditional := make(map[string]model.BundleEntry, len(bundleAdditional.BundleEntries))
	for _, bundleEntry := range bundleAdditional.BundleEntries {
		bundleEntriesAdditional[bundleEntry.NameWithPath] = bundleEntry
	}

	for nameWithPath, bundleEntryExisting := range bundleEntriesExisting {
		bundleEntryAdditional, ok := bundleEntriesAdditional[nameWithPath]
		if ok {
			if bundleEntryAdditional.Hash != bundleEntryExisting.Hash {
				diffEntries = append(diffEntries, DiffEntry{
					Type:       DiffEntryTypeDif,
					Name:       nameWithPath,
					Existing:   bundleEntryExisting,
					Additional: bundleEntryAdditional,
				})
			}
		} else {
			diffEntries = append(diffEntries, DiffEntry{
				Type:     DiffEntryTypeDel,
				Name:     nameWithPath,
				Existing: bundleEntryExisting,
			})
		}
	}
	for nameWithPath, bundleEntryAdditional := range bundleEntriesAdditional {
		_, ok := bundleEntriesExisting[nameWithPath]
		if !ok {
			diffEntries = append(diffEntries, DiffEntry{
				Type:       DiffEntryTypeAdd,
				Name:       nameWithPath,
				Additional: bundleEntryAdditional,
			})
		}
	}
	return BundleDiff{
		Entries: diffEntries,
	}, nil
}
