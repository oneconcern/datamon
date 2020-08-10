package core

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"

	"github.com/oneconcern/datamon/pkg/storage"
	"gopkg.in/yaml.v2"

	"github.com/oneconcern/datamon/pkg/model"
)

// DeleteRepo removes a repository from metadata
func DeleteRepo(repo string, store storage.Store) error {
	if err := RepoExists(repo, store); err != nil {
		return fmt.Errorf("cannot find repo: %s: %v", repo, err)
	}

	// 1. remove all bundles in repo
	bundles, err := ListBundles(repo, store)
	if err != nil {
		return fmt.Errorf("cannot list bundles in repo %s: %v", repo, err)
	}
	for _, b := range bundles {
		if e := DeleteBundle(repo, store, b.ID); e != nil {
			return fmt.Errorf("cannot delete bundle %s in repo %s: %v", b.ID, repo, e)
		}
	}

	pth := model.GetArchivePathToRepoDescriptor(repo)
	if err := store.Delete(context.Background(), pth); err != nil {
		return fmt.Errorf("cannot delete repo: %s: %v", repo, err)
	}
	return nil
}

// DeleteBundle removes a single bundle from a repo
func DeleteBundle(repo string, store storage.Store, bundleID string) error {
	if err := RepoExists(repo, store); err != nil {
		return fmt.Errorf("cannot find repo: %s: %v", repo, err)
	}

	pth := model.GetArchivePathToBundle(repo, bundleID)
	r, err := store.Get(context.Background(), pth)
	if err != nil {
		return fmt.Errorf("cannot rerieve bundle metadata from bundle: %s in repo %s: %v", bundleID, repo, err)
	}
	o, err := ioutil.ReadAll(r)
	if err != nil {
		return fmt.Errorf("cannot read bundle metadata from bundle: %s in repo %s: %v", bundleID, repo, err)
	}
	var bundle model.BundleDescriptor
	err = yaml.Unmarshal(o, &bundle)
	if err != nil {
		return fmt.Errorf("cannot unmarshal bundle metadata from bundle: %s in repo %s: %v", bundleID, repo, err)
	}

	// 1. remove all labels for that bundle
	labels, err := ListLabels(repo, store, "")
	if err != nil {
		return fmt.Errorf("cannot list labels in repo %s: %v", repo, err)
	}

	for _, l := range labels {
		if l.BundleID == bundleID {
			if e := DeleteLabel(repo, store, l.Name); e != nil {
				return fmt.Errorf("cannot delete label %s on bundle %s in repo %s: %v", l.Name, bundleID, repo, e)
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
func DeleteLabel(repo string, store storage.Store, name string) error {
	if err := RepoExists(repo, store); err != nil {
		return fmt.Errorf("cannot find repo: %s: %v", repo, err)
	}
	// TODO(fred): delete all versions???
	pth := model.GetArchivePathToLabel(repo, name)
	if e := store.Delete(context.Background(), pth); e != nil {
		return fmt.Errorf("cannot delete label %s for repo %s: %v", name, repo, e)
	}
	return nil
}

// DeleteEntriesFromRepo remove a list of file entries from all bundles in a repo
func DeleteEntriesFromRepo(repo string, store storage.Store, toDelete []string) error {
	if err := RepoExists(repo, store); err != nil {
		return fmt.Errorf("cannot find repo: %s: %v", repo, err)
	}
	ctx := context.Background()

	// 1. scan all bundles
	bundles, err := ListBundles(repo, store)
	if err != nil {
		return fmt.Errorf("cannot list bundles in repo %s: %v", repo, err)
	}
	for _, b := range bundles {
		bundleID := b.ID
		pth := model.GetArchivePathToBundle(repo, bundleID)
		r, e := store.Get(context.Background(), pth)
		if e != nil {
			return fmt.Errorf("cannot rerieve bundle metadata from bundle: %s in repo %s: %v", bundleID, repo, e)
		}
		o, e := ioutil.ReadAll(r)
		if e != nil {
			return fmt.Errorf("cannot read bundle metadata from bundle: %s in repo %s: %v", bundleID, repo, e)
		}
		var bundle model.BundleDescriptor
		e = yaml.Unmarshal(o, &bundle)
		if e != nil {
			return fmt.Errorf("cannot unmarshal bundle metadata from bundle: %s in repo %s: %v", bundleID, repo, e)
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
