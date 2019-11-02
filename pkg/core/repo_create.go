package core

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	context2 "github.com/oneconcern/datamon/pkg/context"

	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage"
	"gopkg.in/yaml.v2"
)

func CreateRepo(repo model.RepoDescriptor, stores context2.Stores) error {
	store := getRepoStore(stores) // TODO: Integrate with WAL.
	err := model.Validate(repo)
	if err != nil {
		return err
	}
	r, e := yaml.Marshal(repo)
	if e != nil {
		return err
	}
	path := model.GetArchivePathToRepoDescriptor(repo.Name)
	err = store.Put(context.Background(), path, bytes.NewReader(r), storage.NoOverWrite)
	if err != nil {
		if strings.Contains(err.Error(), "googleapi: Error 412: Precondition Failed, conditionNotMet") {
			return fmt.Errorf("repo already exists: %s", repo.Name)
		}
		return err
	}
	return nil
}
