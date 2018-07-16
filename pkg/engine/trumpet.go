package engine

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/oneconcern/trumpet/pkg/store"
	"github.com/oneconcern/trumpet/pkg/store/localfs"
)

const (
	// DefaultBranch to use when none are specified
	DefaultBranch = "master"
	// repos         = "repos"
	stage   = "stage"
	bundles = "bundles"
)

// New initializes a new runtime for trumpet
func New(baseDir string) (*Runtime, error) {
	if baseDir == "" {
		baseDir = ".trumpet"
	}
	repos := localfs.NewRepos(baseDir)
	if err := repos.Initialize(); err != nil {
		return nil, err
	}
	return &Runtime{
		baseDir: baseDir,
		repos:   repos,
	}, nil
}

// Runtime for trumpet
type Runtime struct {
	baseDir string
	repos   store.RepoStore
}

// ListRepo known in the trumpet database
func (r *Runtime) ListRepo() ([]Repo, error) {
	rr, err := r.repos.List()
	if err != nil {
		return nil, err
	}

	repos := make([]Repo, len(rr))
	for i, name := range rr {
		repo, err := r.GetRepo(name)
		if err != nil {
			return nil, err
		}
		repos[i] = *repo
	}
	return repos, nil
}

// GetRepo from trumpet database
func (r *Runtime) GetRepo(name string) (*Repo, error) {
	rr, err := r.repos.Get(name)
	if err != nil {
		return nil, err
	}

	return r.makeRepo(rr.Name, rr.Description, "")
}

func (r *Runtime) makeRepo(name, description, branch string) (*Repo, error) {
	if name == "" {
		return nil, store.NameIsRequired
	}

	if branch == "" {
		branch = DefaultBranch
	}

	bs := localfs.NewBundleStore(filepath.Join(r.baseDir, name, "bundles"))
	if err := bs.Initialize(); err != nil {
		return nil, err
	}

	snapshots := localfs.NewSnapshotStore(filepath.Join(r.baseDir, name, "bundles"))
	if err := snapshots.Initialize(); err != nil {
		return nil, err
	}

	stage, err := newStage(filepath.Join(r.baseDir, name, stage), bs)
	if err != nil {
		return nil, err
	}

	return &Repo{
		Name:          name,
		Description:   description,
		CurrentBranch: branch,
		baseDir:       filepath.Join(r.baseDir, name),
		stage:         stage,
		snapshots:     snapshots,
		bundles:       bs,
	}, nil
}

// CreateRepo creates a repository in the database
func (r *Runtime) CreateRepo(name, description string) (*Repo, error) {
	repo, err := r.makeRepo(name, description, "")
	if err != nil {
		return nil, fmt.Errorf("create repo: %v", err)
	}

	err = r.repos.Create(&store.Repo{
		Name:        name,
		Description: description,
	})
	if err != nil {
		return nil, fmt.Errorf("create repo: %v", err)
	}
	return repo, nil
}

// DeleteRepo removes a repository from trumpet
func (r *Runtime) DeleteRepo(name string) error {
	if err := os.RemoveAll(filepath.Join(r.baseDir, name)); err != nil {
		return err
	}
	return r.repos.Delete(name)
}
