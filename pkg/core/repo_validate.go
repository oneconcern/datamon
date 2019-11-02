package core

import (
	"context"
	"fmt"
	"io/ioutil"

	context2 "github.com/oneconcern/datamon/pkg/context"

	"gopkg.in/yaml.v2"

	"github.com/oneconcern/datamon/pkg/model"
)

func RepoExists(repo string, stores context2.Stores) error {
	store := getRepoStore(stores)
	exists, err := store.Has(context.Background(), model.GetArchivePathToRepoDescriptor(repo))
	if err != nil {
		return fmt.Errorf("repo validation failed: Hit err:%s", err)
	}
	if !exists {
		return fmt.Errorf("repo validation: Repo:%s does not exist", repo)
	}
	return nil
}

func GetRepo(repo string, stores context2.Stores) (*model.RepoDescriptor, error) {
	store := stores.Metadata()
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
