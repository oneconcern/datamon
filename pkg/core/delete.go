package core

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"

	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/storage"
	"gopkg.in/yaml.v2"

	"github.com/oneconcern/datamon/pkg/model"
)

// DeleteRepo removes a repository from metadata
func DeleteRepo(repo string, stores context2.Stores, opts ...DeleteOption) error {
	store := GetRepoStore(stores)
	options := deleteOptionsWithDefaults(opts)

	if !options.skipCheckRepo {
		if err := RepoExists(repo, stores); err != nil {
			return fmt.Errorf("cannot find repo: %s: %v", repo, err)
		}
	}

	// 1. remove all bundles in repo
	bundles, err := ListBundles(repo, stores)
	if err != nil {
		return fmt.Errorf("cannot list bundles in repo %s: %v", repo, err)
	}
	bopts := opts
	bopts = append(bopts,
		WithDeleteSkipCheckRepo(true),
		WithDeleteSkipDeleteLabel(true),
	)

	for _, b := range bundles {
		if e := DeleteBundle(repo, stores, b.ID, bopts...); e != nil {
			return fmt.Errorf("cannot delete bundle %s in repo %s: %v", b.ID, repo, e)
		}
	}

	pth := model.GetArchivePathToRepoDescriptor(repo)
	if err = store.Delete(context.Background(), pth); err != nil {
		return fmt.Errorf("cannot delete repo: %s: %v", repo, err)
	}

	// 2. remove all labels
	labels, err := ListLabels(repo, stores)
	if err != nil {
		return fmt.Errorf("cannot list labels in repo %s: %v", repo, err)
	}

	for _, l := range labels {
		if e := DeleteLabel(repo, stores, l.Name, WithDeleteSkipCheckRepo(true)); e != nil {
			return fmt.Errorf("cannot delete label %s on bundle %s in repo %s: %v", l.Name, l.BundleID, repo, e)
		}
	}

	return nil
}

// DeleteBundle removes a single bundle from a repo.
func DeleteBundle(repo string, stores context2.Stores, bundleID string, opts ...DeleteOption) error {
	options := deleteOptionsWithDefaults(opts)

	if !options.skipCheckRepo {
		if err := RepoExists(repo, stores); err != nil {
			return fmt.Errorf("cannot find repo: %s: %v", repo, err)
		}
	}

	store := getMetaStore(stores)
	pth := model.GetArchivePathToBundle(repo, bundleID)
	bundle, err := downloadBundleDescriptor(store, repo, pth, defaultSettings())
	if err != nil {
		return fmt.Errorf("cannot retrieve bundle metadata from bundle: %s in repo %s: %v", bundleID, repo, err)
	}

	if !options.skipDeleteLabel {
		// 1. remove all labels for that bundle
		labels, err := ListLabels(repo, stores)
		if err != nil {
			return fmt.Errorf("cannot list labels in repo %s: %v", repo, err)
		}

		for _, l := range labels {
			if l.BundleID == bundleID {
				if e := DeleteLabel(repo, stores, l.Name, opts...); e != nil {
					return fmt.Errorf("cannot delete label %s on bundle %s in repo %s: %v", l.Name, bundleID, repo, e)
				}
			}
		}
	}

	// 2. remove all file entry index files for that bundle
	indexFiles := bundle.BundleEntriesFileCount
	for i := uint64(0); i < indexFiles; i++ {
		archivePathToBundleFileList := model.GetArchivePathToBundleFileList(repo, bundleID, i)
		if e := store.Delete(context.Background(), archivePathToBundleFileList); e != nil {
			return fmt.Errorf("cannot delete file list %s on bundle %s in repo %s: %v", archivePathToBundleFileList, bundleID, repo, e)
		}
	}

	// 3. remove bundle descriptor
	if e := store.Delete(context.Background(), pth); e != nil {
		return fmt.Errorf("cannot delete bundle descriptor for %s in repo %s: %v", bundleID, repo, e)
	}
	return nil
}

// DeleteLabel removes a single label from a repo
func DeleteLabel(repo string, stores context2.Stores, name string, opts ...DeleteOption) error {
	options := deleteOptionsWithDefaults(opts)

	if !options.skipCheckRepo {
		if err := RepoExists(repo, stores); err != nil {
			return fmt.Errorf("cannot find repo: %s: %v", repo, err)
		}
	}

	// TODO(fred): delete all versions???
	store := stores.VMetadata()
	pth := model.GetArchivePathToLabel(repo, name)
	if e := store.Delete(context.Background(), pth); e != nil {
		return fmt.Errorf("cannot delete label %s for repo %s: %v", name, repo, e)
	}
	return nil
}

// DeleteEntriesFromRepo remove a list of file entries from all bundles in a repo
func DeleteEntriesFromRepo(repo string, stores context2.Stores, toDelete []string) error {
	if err := RepoExists(repo, stores); err != nil {
		return fmt.Errorf("cannot find repo: %s: %v", repo, err)
	}
	store := getMetaStore(stores)
	ctx := context.Background()

	// 1. scan all bundles
	bundles, err := ListBundles(repo, stores)
	if err != nil {
		return fmt.Errorf("cannot list bundles in repo %s: %v", repo, err)
	}
	for _, b := range bundles {
		bundleID := b.ID
		pth := model.GetArchivePathToBundle(repo, bundleID)
		bundle, e := downloadBundleDescriptor(store, repo, pth, defaultSettings())
		if e != nil {
			return fmt.Errorf("cannot download metadata for bundle %s in repo %s: %v", bundleID, repo, err)
		}

		// 2. scan file lists for that bundle
		indexFiles := bundle.BundleEntriesFileCount
		for i := uint64(0); i < indexFiles; i++ {
			archivePathToBundleFileList := model.GetArchivePathToBundleFileList(repo, bundleID, i)
			rdr, e := store.Get(ctx, archivePathToBundleFileList)
			if e != nil {
				return fmt.Errorf("cannot retrieve file list index %d for bundle %s in repo %s: %v", i, bundleID, repo, e)
			}
			bundleEntriesBuffer, e := ioutil.ReadAll(rdr)
			if e != nil {
				return fmt.Errorf("cannot read file list index %d for bundle %s in repo %s: %v", i, bundleID, repo, e)
			}
			var bundleEntries model.BundleEntries
			e = yaml.Unmarshal(bundleEntriesBuffer, &bundleEntries)
			if e != nil {
				return fmt.Errorf("cannot unmarshal file list index %d for bundle %s in repo %s: %v", i, bundleID, repo, e)
			}

			// 3. scan entries in file list
			newBundleEntry := model.BundleEntries{
				BundleEntries: make([]model.BundleEntry, 0, len(bundleEntries.BundleEntries)),
			}
			listModified := false
			for _, entry := range bundleEntries.BundleEntries {
				entryDeleted := false
				for _, fileToDelete := range toDelete {
					if entry.NameWithPath == fileToDelete {
						entryDeleted = true
						listModified = true
						break
					}
				}
				if !entryDeleted {
					newBundleEntry.BundleEntries = append(newBundleEntry.BundleEntries, entry)
				}
			}
			if listModified {
				// 4. overwrite updated file list
				buffer, erm := yaml.Marshal(newBundleEntry)
				if erm != nil {
					return fmt.Errorf("cannot marshal file list index %d for bundle %s in repo %s: %v", i, bundleID, repo, erm)
				}
				// TODO(fred): make sure overwrite is done without trailing thrash
				erp := store.Put(ctx, archivePathToBundleFileList, bytes.NewReader(buffer), storage.OverWrite)
				if erp != nil {
					return fmt.Errorf("cannot overwrite file list index %d for bundle %s in repo %s: %v", i, bundleID, repo, erp)
				}
			}
		}
	}
	return nil
}
