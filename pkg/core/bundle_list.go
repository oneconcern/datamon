package core

import (
	"context"
	"io/ioutil"
	"strings"

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
	ks, _, _ := store.KeysPrefix(context.Background(), "", model.GetArchivePathPrefixToBundles(repo), "", 1000000)
	var keys = make([]string, 0)
	for _, k := range ks {
		c := strings.SplitN(k, "/", 4)[3]
		if c != "bundle.json" {
			continue
		}
		c = strings.SplitN(k, "/", 4)[2]
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
