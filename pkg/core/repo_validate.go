package core

import (
	"context"
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
)

func RepoExists(repo string, store storage.Store) error {
	exists, err := store.Has(context.Background(), model.GetArchivePathToRepoDescriptor(repo))
	if err != nil {
		return fmt.Errorf("repo validation failed: Hit err:%s", err)
	}
	if !exists {
		return fmt.Errorf("repo validation: Repo:%s does not exist", repo)
	}
	return nil
}

func GetRepo(repo string, store storage.Store) (*model.RepoDescriptor, error) {
	r, err := store.Get(context.Background(), model.GetArchivePathToRepoDescriptor(repo))
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
	return &rd, nil
}
