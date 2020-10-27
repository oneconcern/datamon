package core

import (
	"bytes"
	"context"
	"fmt"
	"hash/crc32"
	"io/ioutil"

	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
)

// RenameRepo renames a repository in metadata
func RenameRepo(repo, newRepo string, stores context2.Stores) error {

	if err := RepoExists(repo, stores); err != nil {
		return fmt.Errorf("cannot find repo: %s: %v", repo, err)
	}

	if err := RepoExists(newRepo, stores); err == nil {
		return fmt.Errorf("new repo %s already exists", newRepo)
	}

	// 1. create new repo
	desc, err := GetRepo(repo, stores)
	if err != nil {
		return fmt.Errorf("cannot retrieve repo metadata for %s: %v", repo, err)
	}

	newDesc := *desc
	newDesc.Name = newRepo
	err = CreateRepo(newDesc, stores)
	if err != nil {
		return fmt.Errorf("cannot create new repo %s: %v", newRepo, err)
	}

	ctx := context.Background()

	// 1. copy all bundle metadata to new repo
	err = ListBundlesApply(repo, stores, func(bundle model.BundleDescriptor) error {
		// 1.1. copy bundle descriptor
		b := NewBundle(ContextStores(stores), BundleDescriptor(&bundle), Repo(newRepo), BundleID(bundle.ID))
		e := uploadBundleDescriptor(ctx, b)
		if e != nil {
			return e
		}

		// 1.2. copy all file lists for this bundle to new repo
		indexFiles := bundle.BundleEntriesFileCount
		for i := uint64(0); i < indexFiles; i++ {
			oldFileList := model.GetArchivePathToBundleFileList(repo, bundle.ID, i)
			rdr, ee := b.MetaStore().Get(ctx, oldFileList)
			if e != nil {
				return ee
			}

			newFileList := model.GetArchivePathToBundleFileList(newRepo, bundle.ID, i)

			msCRC, ok := b.MetaStore().(storage.StoreCRC)
			if ok {
				buffer, eb := ioutil.ReadAll(rdr)
				if eb != nil {
					return eb
				}

				crc := crc32.Checksum(buffer, crc32.MakeTable(crc32.Castagnoli))
				e = msCRC.PutCRC(ctx, newFileList, bytes.NewReader(buffer), storage.NoOverWrite, crc)
			} else {
				e = b.MetaStore().Put(ctx, newFileList, rdr, storage.NoOverWrite)
			}
			if e != nil {
				return e
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("cannot copy bundles in repo %s: %v", repo, err)
	}

	// 3. copy all labels to new repo
	err = ListLabelsApply(repo, stores, func(label model.LabelDescriptor) error {
		l := NewLabel(LabelDescriptor(&label))
		b := NewBundle(ContextStores(stores), BundleID(label.BundleID), Repo(newRepo))
		return l.UploadDescriptor(ctx, b)
	})
	if err != nil {
		return fmt.Errorf("cannot copy labels in repo %s: %v", repo, err)
	}

	err = DeleteRepo(repo, stores)
	if err != nil {
		return fmt.Errorf("new repo has been created, but couldn't remove original repo: %v", err)
	}

	return nil
}
