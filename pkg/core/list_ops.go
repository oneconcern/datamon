package core

import (
	"context"
	"fmt"
	"io/ioutil"

	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/core/lister"
	"github.com/oneconcern/datamon/pkg/core/status"
	"github.com/oneconcern/datamon/pkg/model"
	"gopkg.in/yaml.v2"
)

// ApplyBundleFunc is a function to be applied on a bundle
type ApplyBundleFunc func(model.BundleDescriptor) error

// ApplyRepoFunc is a function to be applied on a repo
type ApplyRepoFunc func(model.RepoDescriptor) error

// ApplyLabelFunc is a function to be applied on a label
type ApplyLabelFunc func(model.LabelDescriptor) error

// ListBundles returns a list of bundle descriptors from a repo. It collects all bundles until completion.
//
// NOTE: this func could become deprecated. At this moment, however, it is used by pkg/web.
func ListBundles(repo string, stores context2.Stores, opts ...Option) (model.BundleDescriptors, error) {
	l := makeBundleLister(repo, stores, opts)
	list, err := l.List()
	result := make(model.BundleDescriptors, 0, len(list))
	for _, e := range list {
		result = append(result, e.(model.BundleDescriptor))
	}
	return result, err // may return partial results when err != nil
}

// ListBundlesApply applies some function to the retrieved bundles, in lexicographic order of keys.
//
// The execution of the applied function does not block background retrieval of more keys and bundle descriptors.
//
// Example usage: printing bundle descriptors as they come
//
//   err := core.ListBundlesApply(repo, store, func(bundle model.BundleDescriptor) error {
//				fmt.Fprintf(os.Stderr, "%v\n", bundle)
//				return nil
//			})
func ListBundlesApply(repo string, stores context2.Stores, apply ApplyBundleFunc, opts ...Option) error {
	l := makeBundleLister(repo, stores, opts)
	return l.ListableApply(func(in lister.Listable) error {
		return apply(in.(model.BundleDescriptor))
	})
}

// ListRepos returns all repos from a store
func ListRepos(stores context2.Stores, opts ...Option) ([]model.RepoDescriptor, error) {
	l := makeRepoLister(stores, opts)
	list, err := l.List()
	result := make(model.RepoDescriptors, 0, len(list))
	for _, e := range list {
		result = append(result, e.(model.RepoDescriptor))
	}
	return result, err
}

// ListReposApply applies some function to the retrieved repos, in lexicographic order of keys.
func ListReposApply(stores context2.Stores, apply ApplyRepoFunc, opts ...Option) error {
	l := makeRepoLister(stores, opts)
	return l.ListableApply(func(in lister.Listable) error { return apply(in.(model.RepoDescriptor)) })
}

// ListLabels returns all labels from a repo
func ListLabels(repo string, stores context2.Stores, prefix string, opts ...Option) ([]model.LabelDescriptor, error) {
	l := makeLabelLister(repo, stores, prefix, opts)
	list, err := l.List()
	result := make(model.LabelDescriptors, 0, len(list))
	for _, e := range list {
		result = append(result, e.(model.LabelDescriptor))
	}
	return result, err
}

// ListLabelsApply applies some function to the retrieved labels, in lexicographic order of keys.
func ListLabelsApply(repo string, stores context2.Stores, prefix string, apply ApplyLabelFunc, opts ...Option) error {
	l := makeLabelLister(repo, stores, prefix, opts)
	return l.ListableApply(func(in lister.Listable) error { return apply(in.(model.LabelDescriptor)) })
}

func makeBundleLister(repo string, stores context2.Stores, extra []Option) *lister.Lister {
	settings := newSettings(extra...)
	store := GetBundleStore(stores)
	opts := []lister.Option{
		lister.Concurrency(settings.concurrentList),
		lister.Checker(func() error {
			return RepoExists(repo, stores)
		}),
		lister.Iterator(func(next string) ([]string, string, error) {
			return store.KeysPrefix(context.Background(), next, model.GetArchivePathPrefixToBundles(repo), "/", settings.batchSize)
		}),
		lister.Downloader(func(key string) (lister.Listable, error) {
			apc, err := model.GetArchivePathComponents(key)
			if err != nil {
				return model.BundleDescriptor{}, err
			}

			r, err := store.Get(context.Background(), model.GetArchivePathToBundle(repo, apc.BundleID))
			if err != nil {
				return model.BundleDescriptor{}, err
			}

			o, err := ioutil.ReadAll(r)
			if err != nil {
				return model.BundleDescriptor{}, err
			}

			var bd model.BundleDescriptor
			err = yaml.Unmarshal(o, &bd)
			if err != nil {
				return model.BundleDescriptor{}, err
			}

			if bd.ID != apc.BundleID {
				err = fmt.Errorf("bundle IDs in descriptor '%v' and archive path '%v' don't match", bd.ID, apc.BundleID)
				return model.BundleDescriptor{}, err
			}
			return bd, nil
		}),
	}
	return lister.New(opts...)
}

func makeRepoLister(stores context2.Stores, extra []Option) *lister.Lister {
	settings := newSettings(extra...)
	store := GetRepoStore(stores)
	opts := []lister.Option{
		lister.Concurrency(settings.concurrentList),
		lister.Iterator(func(next string) ([]string, string, error) {
			return store.KeysPrefix(context.Background(), next, model.GetArchivePathPrefixToRepos(), "", settings.batchSize)
		}),
		lister.Downloader(func(key string) (lister.Listable, error) {
			apc, err := model.GetArchivePathComponents(key)
			if err != nil {
				return model.RepoDescriptor{}, err
			}

			repo := model.GetArchivePathToRepoDescriptor(apc.Repo)
			has, err := store.Has(context.Background(), repo)
			if err != nil {
				return model.RepoDescriptor{}, err
			}
			if !has {
				return model.RepoDescriptor{}, status.ErrNotFound
			}

			r, err := store.Get(context.Background(), repo)
			if err != nil {
				return model.RepoDescriptor{}, err
			}

			o, err := ioutil.ReadAll(r)
			if err != nil {
				return model.RepoDescriptor{}, err
			}

			var rd model.RepoDescriptor
			err = yaml.Unmarshal(o, &rd)
			if err != nil {
				return model.RepoDescriptor{}, err
			}
			if rd.Name != apc.Repo {
				return rd, fmt.Errorf("repo names in descriptor '%v' and archive path '%v' don't match", rd.Name, apc.Repo)
			}
			return rd, nil
		}),
	}
	return lister.New(opts...)
}

func makeLabelLister(repo string, stores context2.Stores, prefix string, extra []Option) *lister.Lister {
	settings := newSettings(extra...)
	store := GetLabelStore(stores)
	opts := []lister.Option{
		lister.Concurrency(settings.concurrentList),
		lister.Iterator(func(next string) ([]string, string, error) {
			return store.KeysPrefix(context.Background(), next, model.GetArchivePathPrefixToLabels(repo, prefix), "", settings.batchSize)
		}),
		lister.Downloader(func(key string) (lister.Listable, error) {
			apc, err := model.GetArchivePathComponents(key)
			if err != nil {
				return model.LabelDescriptor{}, err
			}

			bundle := NewBundle(Repo(repo), ContextStores(stores))
			labelName := apc.LabelName
			label := NewLabel(
				LabelDescriptor(model.NewLabelDescriptor(model.LabelName(labelName))),
			)
			if err = label.DownloadDescriptor(context.Background(), bundle, false); err != nil {
				return model.LabelDescriptor{}, err
			}

			if label.Descriptor.Name == "" {
				label.Descriptor.Name = apc.LabelName
			} else if label.Descriptor.Name != apc.LabelName {
				return model.LabelDescriptor{}, err
			}
			return label.Descriptor, nil
		}),
	}
	return lister.New(opts...)
}
