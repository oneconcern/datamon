package core

import (
	"bytes"
	"context"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	"gopkg.in/yaml.v2"
)

func CreateRepo(repo model.RepoDescriptor, store storage.Store) error {
	err := model.Validate(repo)
	if err != nil {
		return err
	}
	r, e := yaml.Marshal(repo)
	if e != nil {
		return err
	}
	path := model.GetArchivePathToRepoDescriptor(repo.Name)
	err = store.Put(context.Background(), path, bytes.NewReader(r), storage.IfNotPresent)
	if err != nil {
		return err
	}
	return nil
}
