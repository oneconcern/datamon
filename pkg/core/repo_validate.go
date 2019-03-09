package core

import (
	"context"
	"fmt"

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
