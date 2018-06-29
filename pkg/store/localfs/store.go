package localfs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/json-iterator/go"
	"github.com/oneconcern/trumpet/pkg/store"
	"github.com/spf13/afero"
	"github.com/teris-io/shortid"
)

const (
	repoDb   = "repos"
	blobsDb  = "blobs"
	modelsDb = "models"
	runsDb   = "runs"
)

var (
	_ store.RepoStore = &localFsStore{}
)

func BaseDir(path string) Option {
	return func(fs *localFsStore) {
		fs.baseDir = path
	}
}

func FileSystem(afs afero.Fs) Option {
	return func(fs *localFsStore) {
		fs.fs = afs
	}
}

type Option func(fs *localFsStore)

func New(opts ...Option) *localFsStore {
	fs := &localFsStore{
		baseDir: ".trumpet",
		fs:      afero.NewOsFs(),
	}
	for _, configure := range opts {
		configure(fs)
	}
	return fs
}

type localFsStore struct {
	baseDir string
	repos
	fs   afero.Fs
	once sync.Once
}

func (fs *localFsStore) Initialize() error {
	fs.once.Do(func() {
		fs.repos = repos{fs: fs.fs, BaseDir: filepath.Join(fs.baseDir, repoDb)}
	})
	return nil
}

func (fs *localFsStore) Close() error {
	return fs.repos.Close()
}

type repos struct {
	fs      afero.Fs
	BaseDir string
}

func dbExists(dbLoc string) bool {
	if _, err := os.Stat(dbLoc); os.IsNotExist(err) {
		return false
	}
	return true
}

func (r *repos) List() ([]string, error) {
	var names []string
	matches, err := filepath.Glob(fmt.Sprintf("%s/*.json", r.BaseDir))
	if err != nil {
		return nil, err
	}

	for _, match := range matches {
		cfn := r.cleanRepoPath(match)
		if cfn != "" {
			names = append(names, cfn)
		}
	}
	return names, nil
}

func (r *repos) cleanRepoPath(pth string) string {
	fname := filepath.Base(pth)
	if filepath.Ext(fname) == ".json" {
		return fname[:len(fname)-5]
	}
	return ""
}

func (r *repos) repoPath(name string) string {
	return filepath.Join(r.BaseDir, name+".json")
}

func (r *repos) Get(name string) (*store.Repo, error) {
	rPath := r.repoPath(name)
	if !dbExists(rPath) {
		return nil, fmt.Errorf("%v: %q", store.RepoNotFound, name)
	}

	fi, err := r.fs.Open(rPath)
	if err != nil {
		return nil, err
	}
	defer fi.Close()

	var result store.Repo
	dec := jsoniter.NewDecoder(fi)
	if err := dec.Decode(&result); err != nil {
		return nil, err
	}

	if err := fi.Close(); err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *repos) Create(repo *store.Repo) error {
	if strings.TrimSpace(repo.Name) == "" {
		return store.NameIsRequired
	}

	rPath := r.repoPath(repo.Name)
	if dbExists(rPath) {
		return fmt.Errorf("%v: %q", store.RepoAlreadyExists, repo.Name)
	}

	return r.writeRepo(repo.Name, repo)
}

func (r *repos) writeRepo(fname string, repo *store.Repo) error {
	rPath := r.repoPath(fname)
	fi, err := r.fs.OpenFile(rPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("create record for %q: %v", repo.Name, err)
	}
	defer fi.Close()

	enc := jsoniter.NewEncoder(fi)
	if err := enc.Encode(repo); err != nil {
		return fmt.Errorf("writing data for %q: %v", repo.Name, err)
	}
	return fi.Close()
}

func (r *repos) Update(repo *store.Repo) error {
	if !dbExists(r.repoPath(repo.Name)) {
		return fmt.Errorf("%v: %s", store.RepoNotFound, repo.Name)
	}
	id, err := shortid.Generate()
	if err != nil {
		return err
	}

	fname := fmt.Sprintf("%s-%s", repo.Name, id)
	fpath := r.repoPath(fname)
	defer os.Remove(fpath) // ensure that the temp file is gone, even on error somewhere
	if err := r.writeRepo(fname, repo); err != nil {
		return err
	}

	return os.Rename(fpath, r.repoPath(repo.Name))
}

func (r *repos) Delete(name string) error {
	if err := r.fs.Remove(r.repoPath(name)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing %q: %v", name, err)
	}
	return nil
}

func (r *repos) Close() error {
	return nil
}
