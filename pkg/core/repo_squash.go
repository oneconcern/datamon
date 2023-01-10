package core

import (
	"fmt"

	"github.com/blang/semver"
	context2 "github.com/oneconcern/datamon/pkg/context"
)

func RepoSquash(stores context2.Stores, repoName string, opts ...Option) error {
	opts = append(opts, WithMinimalBundle(true)) // limits I/Os with remote store: we only need keys

	bundles, err := ListBundles(repoName, stores, opts...)
	if err != nil {
		return err
	}

	settings := defaultSettings()
	for _, bApply := range opts {
		bApply(&settings)
	}
	if settings.retainNLatest == 0 {
		settings.retainNLatest = 1
	}

	if len(bundles) < settings.retainNLatest+1 {
		// nothing to be squashed
		return nil
	}

	labelsIndex := make(map[string]struct{}, 10)
	if settings.retainTags || settings.retainSemverTags {
		labels, erl := ListLabels(repoName, stores, opts...)
		if erl != nil {
			return erl
		}

		for _, label := range labels {
			if settings.retainTags {
				// any tag is retained
				labelsIndex[label.BundleID] = struct{}{}

				continue
			}

			// only semver tags are retained
			_, ers := semver.ParseTolerant(label.Name)
			if ers == nil {
				labelsIndex[label.BundleID] = struct{}{}
			}
		}
	}

	/*
		// Disabled since we no longer retrieve the actual timestamp. It is okay for all practical purposes.
		//
		// bundles are ordered from oldest to most recent (with natural ksuid ordering).
		// However, ksuid is imperfect when timings differ only slightly (e.g. when running tests).
		// Hence the explicit re-sorting on a slice that is essentially already almost sorted.
		sort.SliceStable(bundles, func(i, j int) bool {
			return bundles[i].Timestamp.Before(bundles[j].Timestamp)
		})
	*/
	deletedIndex := make(map[string]struct{}, len(bundles)-settings.retainNLatest)

	for _, bundle := range bundles[:len(bundles)-settings.retainNLatest] {
		if settings.retainTags || settings.retainSemverTags {
			if _, retain := labelsIndex[bundle.ID]; retain {
				continue
			}
		}

		if erd := DeleteBundle(repoName, stores, bundle.ID,
			// disable some deletion to make it faster, esp. when there are a lot of labels
			WithDeleteSkipCheckRepo(true),
			WithDeleteSkipDeleteLabel(true),
		); erd != nil {
			return erd
		}
		deletedIndex[bundle.ID] = struct{}{}
	}

	// now address labels to keep consistent metadata
	labels, err := ListLabels(repoName, stores)
	if err != nil {
		return fmt.Errorf("cannot list labels in repo %s: %v", repoName, err)
	}

	for _, l := range labels {
		if _, found := deletedIndex[l.BundleID]; !found {
			continue
		}

		if e := DeleteLabel(repoName, stores, l.Name, WithDeleteSkipCheckRepo(true)); e != nil {
			return fmt.Errorf("cannot delete label %s on bundle %s in repo %s: %v", l.Name, l.BundleID, repoName, e)
		}
	}

	return nil
}
