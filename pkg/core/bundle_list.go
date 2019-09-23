package core

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
)

const (
	maxBundlesToList = 1000000
)

// TODO: return a paginated list of id<->bd (separate function in repo_list.go)
func ListBundles(repo string, store storage.Store) ([]model.BundleDescriptor, error) {
	// Get a list
	e := RepoExists(repo, store)
	if e != nil {
		return nil, e
	}
	ks, _, err := store.KeysPrefix(context.Background(), "", model.GetArchivePathPrefixToBundles(repo), "/", maxBundlesToList)
	if err != nil {
		return nil, err
	}
	bds := make([]model.BundleDescriptor, 0)
	for _, k := range ks {
		apc, err := model.GetArchivePathComponents(k)
		if err != nil {
			return nil, err
		}
		r, err := store.Get(context.Background(), model.GetArchivePathToBundle(repo, apc.BundleID))
		if err != nil {
			if strings.Contains(err.Error(), "object doesn't exist") {
				continue
			}
			return nil, err
		}
		o, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		var bd model.BundleDescriptor
		err = yaml.Unmarshal(o, &bd)
		if err != nil {
			return nil, err
		}
		if bd.ID != apc.BundleID {
			return nil, fmt.Errorf("bundle IDs in descriptor '%v' and archive path '%v' don't match",
				bd.ID, apc.BundleID)
		}
		bds = append(bds, bd)
	}

	return bds, nil
}

func GetLatestBundle(repo string, store storage.Store) (string, error) {
	e := RepoExists(repo, store)
	if e != nil {
		return "", e
	}
	ks, _, err := store.KeysPrefix(context.Background(), "", model.GetArchivePathPrefixToBundles(repo), "", 1000000)
	if err != nil {
		return "", err
	}
	if len(ks) == 0 {
		return "", fmt.Errorf("no bundles uploaded to repo: %s", repo)
	}

	apc, err := model.GetArchivePathComponents(ks[len(ks)-1])
	if err != nil {
		return "", err
	}

	return apc.BundleID, nil
}
