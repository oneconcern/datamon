package core

import (
	"context"
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
)

func ListBundles(repo string, store storage.Store) ([]string, error) {
	// TODO: Don;t format string here, return a paginated list of id<->bd
	// Get a list
	e := RepoExists(repo, store)
	if e != nil {
		return nil, e
	}
	ks, _, err := store.KeysPrefix(context.Background(), "", model.GetArchivePathPrefixToBundles(repo), "", 1000000)
	if err != nil {
		return nil, err
	}
	var keys = make([]string, 0)
	for _, k := range ks {
		apc, err := model.GetArchivePathComponents(k)
		if err != nil {
			return nil, err
		}
		if apc.ArchiveFileName != "bundle.json" {
			continue
		}
		c := apc.BundleID
		r, err := store.Get(context.Background(), model.GetArchivePathToBundle(repo, c))
		if err != nil {
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
		keys = append(keys, c+" , "+bd.Timestamp.String()+" , "+bd.Message)
	}

	return keys, nil
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
