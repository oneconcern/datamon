package core

import (
	"context"
	"fmt"
	"io/ioutil"

	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/core/status"
	"github.com/oneconcern/datamon/pkg/model"
	"gopkg.in/yaml.v2"
)

// TODO(fred): remove that file entirely. This is left
// temporarily in order to avoid mixing up file layout changes in
// the same PR.

// GetRepoDescriptorByRepoName returns the descriptor of a named repo
func GetRepoDescriptorByRepoName(stores context2.Stores, repoName string) (model.RepoDescriptor, error) {
	return getRepoDescriptorByRepoName(stores, repoName)
}

func getRepoDescriptorByRepoName(stores context2.Stores, repoName string) (model.RepoDescriptor, error) {
	var rd model.RepoDescriptor
	store := GetRepoStore(stores) // TODO: ReadLog integration
	archivePathToRepoDescriptor := model.GetArchivePathToRepoDescriptor(repoName)
	has, err := GetRepoStore(stores).Has(context.Background(), archivePathToRepoDescriptor)
	if err != nil {
		return rd, err
	}
	if !has {
		return rd, status.ErrNotFound
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
