package core

import (
	"context"
	"io/ioutil"
	"strings"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	yaml "gopkg.in/yaml.v2"
)

func ListRepos(store storage.Store) ([]string, error) {
	// TODO: Don;t format string here, return a paginated list of id<->bd
	// Get a list
	ks, _, _ := store.KeysPrefix(context.Background(), "", model.GetArchivePathPrefixToRepos(), "", 1000000)
	var keys = make([]string, 0)
	for _, k := range ks {
		c := strings.SplitN(k, "/", 3)[1]
		r, err := store.Get(context.Background(), model.GetArchivePathToRepoDescriptor(c))
		if err != nil {
			return nil, err
		}
		o, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		var rd model.RepoDescriptor
		err = yaml.Unmarshal(o, &rd)
		if err != nil {
			return nil, err
		}
		keys = append(keys, c+" , "+rd.Description+" , "+rd.Contributor.Name+" , "+rd.Contributor.Email+" , "+rd.Timestamp.String())
	}

	return keys, nil
}
