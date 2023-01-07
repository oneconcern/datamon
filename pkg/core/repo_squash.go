package core

import (
	"github.com/blang/semver"
	context2 "github.com/oneconcern/datamon/pkg/context"
)

func RepoSquash(stores context2.Stores, repoName string, opts ...Option) error {
	opts = append(opts, WithMinimalBundle(true))

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
	for _, bundle := range bundles[:len(bundles)-settings.retainNLatest] {
		if settings.retainTags || settings.retainSemverTags {
			if _, retain := labelsIndex[bundle.ID]; retain {
				continue
			}
		}

		if erd := DeleteBundle(repoName, stores, bundle.ID); erd != nil {
			return erd
		}
	}

	return nil
}
