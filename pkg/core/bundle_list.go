package core

import (
	"context"
	"fmt"

	context2 "github.com/oneconcern/datamon/pkg/context"
	"github.com/oneconcern/datamon/pkg/model"
)

// TODO(fred): remove that file entirely. This is left
// temporarily in order to avoid mixing up file layout changes in
// the same PR.

const (
	maxMetaFilesToProcess = 1000000
)

// GetLatestBundle returns the latest bundle descriptor from a repo
func GetLatestBundle(repo string, stores context2.Stores) (string, error) {
	e := RepoExists(repo, stores)
	if e != nil {
		return "", e
	}
	ks, _, err := getMetaStore(stores).KeysPrefix(context.Background(), "", model.GetArchivePathPrefixToBundles(repo), "", maxMetaFilesToProcess)
	if err != nil {
		return "", err
	}
	if len(ks) == 0 {
		return "", fmt.Errorf("no bundles uploaded to repo: %s", repo)
	}

	apc, err := model.GetArchivePathComponents(ks[len(ks)-1])
	if err != nil {
		return "", err
	}

	return apc.BundleID, nil
}
