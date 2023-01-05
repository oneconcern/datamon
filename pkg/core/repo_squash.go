package core

import (
	"github.com/blang/semver"
	context2 "github.com/oneconcern/datamon/pkg/context"
)

func RepoSquash(stores context2.Stores, repoName string, opts ...Option) error {
	bundles, err := ListBundles(repoName, stores, opts...)
	if err != nil {
		return err
	}

	if len(bundles) < 2 {
		// nothing to be squashed
		return nil
	}

	settings := defaultSettings()
	for _, bApply := range opts {
		bApply(&settings)
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

	// bundles are ordered from oldest to most recent (with natural ksuid ordering)
	for _, bundle := range bundles[:len(bundles)-1] {
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
