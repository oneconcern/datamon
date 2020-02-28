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

// CreateRepo persists a repository with a repo descriptor and some context's stores
func CreateRepo(repo model.RepoDescriptor, stores context2.Stores) error {
	// TODO(fred): refact options etc to expose a consistent interface, plus support metrics
	store := GetRepoStore(stores) // TODO: Integrate with WAL.
	err := model.ValidateRepo(repo)
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
