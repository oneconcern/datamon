package core

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	yaml "gopkg.in/yaml.v2"
)

const (
	maxReposToList = 1000000
)

// todo: dedupe ListReposPaginated()
func ListRepos(store storage.Store) ([]model.RepoDescriptor, error) {
	// Get a list
	ks, _, _ := store.KeysPrefix(context.Background(), "", model.GetArchivePathPrefixToRepos(), "", maxReposToList)
	var repos = make([]model.RepoDescriptor, 0)
	for _, k := range ks {
		apc, err := model.GetArchivePathComponents(k)
		if err != nil {
			return nil, err
		}
		r, err := store.Get(context.Background(), model.GetArchivePathToRepoDescriptor(apc.Repo))
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
		if rd.Name != apc.Repo {
			return nil, fmt.Errorf("repo names in descriptor '%v' and archive path '%v' don't match",
				rd.Name, apc.Repo)
		}
		repos = append(repos, rd)
	}
	return repos, nil
}

func ListReposPaginated(store storage.Store, token string) ([]model.RepoDescriptor, error) {
	// Get a list
	ks, _, _ := store.KeysPrefix(context.Background(), "", model.GetArchivePathPrefixToRepos(), "", maxReposToList)
	var repos = make([]model.RepoDescriptor, 0)
	tokenHit := false
	for _, k := range ks {
		apc, err := model.GetArchivePathComponents(k)
		if err != nil {
			return nil, err
		}
		if apc.Repo == token {
			tokenHit = true
		}
		if !tokenHit {
			continue
		}
		r, err := store.Get(context.Background(), model.GetArchivePathToRepoDescriptor(apc.Repo))
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
		if rd.Name != apc.Repo {
			return nil, fmt.Errorf("repo names in descriptor '%v' and archive path '%v' don't match",
				rd.Name, apc.Repo)
		}
		repos = append(repos, rd)
	}
	return repos, nil
}
