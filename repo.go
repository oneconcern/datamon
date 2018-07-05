package trumpet

import (
	"os"
	"path/filepath"
	"strings"

	stg "github.com/oneconcern/trumpet/pkg/blob"

	"github.com/oneconcern/trumpet/pkg/store"
)

// Repo is the object that manages repositories
type Repo struct {
	Name          string
	Description   string
	CurrentBranch string

	baseDir string
	stage   *Stage
	bundles store.BundleStore
	objects stg.Store
}

// Stage to record pending changes into
func (r *Repo) Stage() *Stage {
	return r.stage
}

// ListCommits gets the bundles associated with the top level commits
func (r *Repo) ListCommits() ([]store.Bundle, error) {
	return r.bundles.ListTopLevel()
}

// CreateCommit the content of the stage to permanent storage
func (r *Repo) CreateCommit(message, branch string) (hash string, empty bool, err error) {
	if strings.TrimSpace(branch) == "" {
		branch = r.CurrentBranch
	}
	return r.commit(message, branch, "")
}

func (r *Repo) commit(message, branch, snapshot string) (hash string, empty bool, err error) {
	parents, err := r.bundles.ListTopLevelIDs()
	if err != nil {
		return "", true, err
	}

	changes, err := r.Stage().Status()
	if err != nil {
		return "", true, err
	}

	hash, empty, err = r.bundles.Create(message, branch, snapshot, parents, changes)
	if err != nil {
		return "", true, err
	}
	if empty {
		return "", true, nil
	}

	tgtDir := filepath.Join(r.baseDir, "bundles", "objects")
	srcDir := filepath.Join(r.baseDir, "stage", "objects")
	filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		rp, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		tgtPth := filepath.Join(tgtDir, rp)

		os.MkdirAll(filepath.Dir(tgtPth), 0700)
		return os.Rename(path, tgtPth)
	})

	if err = r.Stage().Clear(); err != nil {
		return "", false, err
	}

	return hash, false, nil
}
