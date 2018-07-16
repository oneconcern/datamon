package engine

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	stg "github.com/oneconcern/trumpet/pkg/blob"

	"github.com/oneconcern/trumpet/pkg/store"
)

// NewBundle contains the information about a newly created bundle
type NewBundle struct {
	ID       string `json:"id" yaml:"id"`
	Snapshot string `json:"snapshot" yaml:"snapshot"`
	Branch   string `json:"branch" yaml:"branch"`
	IsEmpty  bool   `json:"is_empty" yaml:"is_empty"`
}

// Repo is the object that manages repositories
type Repo struct {
	Name          string `json:"name" yaml:"name"`
	Description   string `json:"description" yaml:"description"`
	CurrentBranch string `json:"branch" yaml:"branch"`

	baseDir   string
	stage     *Stage
	bundles   store.BundleStore
	snapshots store.SnapshotStore
	objects   stg.Store
}

// Stage to record pending changes into
func (r *Repo) Stage() *Stage {
	return r.stage
}

// ListBranches returns the list of known branches for a given repo
func (r *Repo) ListBranches() ([]string, error) {
	return r.bundles.ListBranches()
}

// ListTags returns the list of known branches for a given repo
func (r *Repo) ListTags() ([]string, error) {
	return r.bundles.ListTags()
}

// ListCommits gets the bundles associated with the top level commits
func (r *Repo) ListCommits() ([]store.Bundle, error) {
	return r.bundles.ListTopLevel()
}

// CreateCommit the content of the stage to permanent storage
func (r *Repo) CreateCommit(message, branch string) (result NewBundle, err error) {
	if strings.TrimSpace(branch) == "" {
		branch = r.CurrentBranch
	}
	return r.commit(message, branch)
}

func (r *Repo) commit(message, branch string) (result NewBundle, err error) {
	result.Branch = branch
	result.IsEmpty = true

	parents, err := r.bundles.ListTopLevelIDs()
	if err != nil {
		return result, err
	}

	changes, err := r.Stage().Status()
	if err != nil {
		return result, err
	}

	hash, empty, err := r.bundles.Create(message, branch, "", parents, changes)
	if err != nil {
		return result, err
	}
	if empty {
		return result, nil
	}
	result.IsEmpty = false

	bundle, err := r.bundles.Get(hash)
	if err != nil {
		return result, err
	}
	result.ID = bundle.ID

	snapshot, err := r.snapshots.Create(bundle)
	if err != nil {
		return result, err
	}
	result.Snapshot = snapshot.ID

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
		return result, err
	}

	return result, nil
}

// Checkout gets the working directory layout
func (r *Repo) Checkout(branch, commit string) (*store.Snapshot, error) {
	var err error
	if branch == "" {
		branch = r.CurrentBranch
	}

	if commit == "" {
		commit, err = r.bundles.HashForBranch(branch)
		if err != nil {
			if !strings.Contains(err.Error(), "not found") {
				return nil, err
			}
			commit, err = r.bundles.HashForTag(branch)
			if err != nil {
				return nil, err
			}
		}
		if commit == "empty" {
			return &store.Snapshot{}, nil
		}
	}

	b, err := r.bundles.Get(commit)
	if err != nil {
		return nil, err
	}

	return r.snapshots.GetForBundle(b.ID)
}

// CreateBranch with the given name, when top level is true
// the branch will be created without a bundle attached to it
func (r *Repo) CreateBranch(name string, topLevel bool) error {
	parent := r.CurrentBranch
	if topLevel {
		parent = ""
	}
	return r.bundles.CreateBranch(parent, name)
}

// DeleteBranch with the given name.
// This will remove all the orphaned data as well as the branch itself
func (r *Repo) DeleteBranch(name string) error {
	if name == "" {
		return errors.New("branch name is required for deleting")
	}
	if name == r.CurrentBranch {
		return errors.New("can't delete the current branch")
	}
	return r.bundles.DeleteBranch(name)
}

// CreateTag with the given name
func (r *Repo) CreateTag(name string) error {
	return r.bundles.CreateTag(r.CurrentBranch, name)
}

// DeleteTag with the given name.
func (r *Repo) DeleteTag(name string) error {
	if name == "" {
		return errors.New("tag name is required for deleting")
	}
	return r.bundles.DeleteTag(name)
}
