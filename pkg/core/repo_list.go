package core

import (
	"context"
	"fmt"
	"io/ioutil"

	context2 "github.com/oneconcern/datamon/pkg/context"

	"github.com/oneconcern/datamon/pkg/model"
	"gopkg.in/yaml.v2"
)

const (
	maxReposToList = 1000000
)

func GetRepoDescriptorByRepoName(stores context2.Stores, repoName string) (model.RepoDescriptor, error) {
	var rd model.RepoDescriptor
	store := getRepoStore(stores) // TODO: ReadLog integration
	archivePathToRepoDescriptor := model.GetArchivePathToRepoDescriptor(repoName)
	has, err := store.Has(context.Background(), archivePathToRepoDescriptor)
	if err != nil {
		return rd, err
	}
	if !has {
		return rd, ErrNotFound
	}
	r, err := store.Get(context.Background(), archivePathToRepoDescriptor)
	if err != nil {
		return rd, err
	}
	o, err := ioutil.ReadAll(r)
	if err != nil {
		return rd, err
	}
	err = yaml.Unmarshal(o, &rd)
	if err != nil {
		return rd, err
	}
	if rd.Name != repoName {
		return rd, fmt.Errorf("repo names in descriptor '%v' and archive path '%v' don't match",
			rd.Name, repoName)
	}
	return rd, nil
}

func ListRepos(stores context2.Stores) ([]model.RepoDescriptor, error) {
	// Get a list
	store := getRepoStore(stores)
	ks, _, _ := store.KeysPrefix(context.Background(), "", model.GetArchivePathPrefixToRepos(), "", maxReposToList)
	var repos = make([]model.RepoDescriptor, 0)
	for _, k := range ks {
		apc, err := model.GetArchivePathComponents(k)
		if err != nil {
			return nil, err
		}

		var rd model.RepoDescriptor
		rd, err = GetRepoDescriptorByRepoName(stores, apc.Repo)
		if err != nil {
			return nil, err
		}
		repos = append(repos, rd)
	}
	return repos, nil
}

// todo: use storage.Store pagination
func ListReposPaginated(stores context2.Stores, token string) ([]model.RepoDescriptor, error) {
	// Get a list
	ks, _, _ := getRepoStore(stores).KeysPrefix(context.Background(), "", model.GetArchivePathPrefixToRepos(), "", maxReposToList)
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

		var rd model.RepoDescriptor
		rd, err = GetRepoDescriptorByRepoName(stores, apc.Repo)
		if err != nil {
			return nil, err
		}
		repos = append(repos, rd)
	}
	return repos, nil
}
