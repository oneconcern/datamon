package engine

import (
	"context"
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
func (r *Repo) ListBranches(ctx context.Context) ([]string, error) {
	return r.bundles.ListBranches(ctx)
}

// ListTags returns the list of known branches for a given repo
func (r *Repo) ListTags(ctx context.Context) ([]string, error) {
	return r.bundles.ListTags(ctx)
}

// ListCommits gets the bundles associated with the top level commits
func (r *Repo) ListCommits(ctx context.Context) ([]store.Bundle, error) {
	return r.bundles.ListTopLevel(ctx)
}

// CreateCommit the content of the stage to permanent storage
func (r *Repo) CreateCommit(ctx context.Context, message, branch string) (result NewBundle, err error) {
	if strings.TrimSpace(branch) == "" {
		branch = r.CurrentBranch
	}
	return r.commit(ctx, message, branch)
}

func (r *Repo) CommitFromChangeSet(ctx context.Context, message, branch string, changes store.ChangeSet) (result NewBundle, err error) {
	result.Branch = branch
	result.IsEmpty = true

	parents, err := r.bundles.ListTopLevelIDs(ctx)
	if err != nil {
		return result, err
	}

	hash, empty, err := r.bundles.Create(ctx, message, branch, "", parents, changes)
	if err != nil {
		return result, err
	}
	if empty {
		return result, nil
	}
	result.IsEmpty = false

	bundle, err := r.bundles.Get(ctx, hash)
	if err != nil {
		return result, err
	}
	result.ID = bundle.ID

	snapshot, err := r.snapshots.Create(ctx, bundle)
	if err != nil {
		return result, err
	}
	result.Snapshot = snapshot.ID

	// TODO: actually upload the files prior to returnin
	return result, nil
}

func (r *Repo) commit(ctx context.Context, message, branch string) (result NewBundle, err error) {
	result.Branch = branch
	result.IsEmpty = true

	parents, err := r.bundles.ListTopLevelIDs(ctx)
	if err != nil {
		return result, err
	}

	changes, err := r.Stage().Status(ctx)
	if err != nil {
		return result, err
	}

	hash, empty, err := r.bundles.Create(ctx, message, branch, "", parents, changes)
	if err != nil {
		return result, err
	}
	if empty {
		return result, nil
	}
	result.IsEmpty = false

	bundle, err := r.bundles.Get(ctx, hash)
	if err != nil {
		return result, err
	}
	result.ID = bundle.ID

	snapshot, err := r.snapshots.Create(ctx, bundle)
	if err != nil {
		return result, err
	}
	result.Snapshot = snapshot.ID

	// TODO: make this a batch job
	srcDir := filepath.Join(r.baseDir, "stage", "objects")
	filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		rp, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		f, err := os.Open(info.Name())
		if err != nil {
			return err
		}
		defer f.Close()

		if err := r.objects.Put(ctx, rp, f); err != nil {
			return err
		}
		return f.Close()
	})

	if err = r.Stage().Clear(ctx); err != nil {
		return result, err
	}

	return result, nil
}

// Checkout gets the working directory layout
func (r *Repo) Checkout(ctx context.Context, branch, commit string) (*store.Snapshot, error) {
	var err error
	if branch == "" {
		branch = r.CurrentBranch
	}

	if commit == "" {
		commit, err = r.bundles.HashForBranch(ctx, branch)
		if err != nil {
			if !strings.Contains(err.Error(), "not found") {
				return nil, err
			}
			commit, err = r.bundles.HashForTag(ctx, branch)
			if err != nil {
				return nil, err
			}
		}
		if commit == "empty" {
			return &store.Snapshot{}, nil
		}
	}

	b, err := r.bundles.Get(ctx, commit)
	if err != nil {
		return nil, err
	}

	return r.snapshots.GetForBundle(ctx, b.ID)
}

// GetBundle for the specified commit id or name
func (r *Repo) GetBundle(ctx context.Context, commit string) (*store.Bundle, error) {
	var err error
	branch := commit
	if commit == "" {
		branch = r.CurrentBranch
	}
	commit, err = r.bundles.HashForBranch(ctx, branch)
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			return nil, err
		}

		commit, err = r.bundles.HashForTag(ctx, branch)
		if err != nil {
			if !strings.Contains(err.Error(), "not found") {
				return nil, err
			}
		}
		if commit == "empty" {
			return &store.Bundle{}, nil
		}
	}
	return r.bundles.Get(ctx, commit)
}

// CreateBranch with the given name, when top level is true
// the branch will be created without a bundle attached to it
func (r *Repo) CreateBranch(ctx context.Context, name string, topLevel bool) error {
	parent := r.CurrentBranch
	if topLevel {
		parent = ""
	}
	return r.bundles.CreateBranch(ctx, parent, name)
}

// DeleteBranch with the given name.
// This will remove all the orphaned data as well as the branch itself
func (r *Repo) DeleteBranch(ctx context.Context, name string) error {
	if name == "" {
		return errors.New("branch name is required for deleting")
	}
	if name == r.CurrentBranch {
		return errors.New("can't delete the current branch")
	}
	return r.bundles.DeleteBranch(ctx, name)
}

// CreateTag with the given name
func (r *Repo) CreateTag(ctx context.Context, name string) error {
	return r.bundles.CreateTag(ctx, r.CurrentBranch, name)
}

// DeleteTag with the given name.
func (r *Repo) DeleteTag(ctx context.Context, name string) error {
	if name == "" {
		return errors.New("tag name is required for deleting")
	}
	return r.bundles.DeleteTag(ctx, name)
}
