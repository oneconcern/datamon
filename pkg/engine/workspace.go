package engine

import (
	"context"
	"os"
	"time"

	"github.com/oneconcern/trumpet/pkg/cafs"

	"github.com/oneconcern/trumpet/pkg/store"
	"github.com/spf13/afero"
)

type Workspace struct {
	dir    string
	head   string
	branch string
	// repo  *Repo
	stage *Stage
	fs    afero.Fs
}

func (ws *Workspace) Status(ctx context.Context) (WorkspaceStatus, error) {
	cs, err := ws.stage.Status(ctx)
	if err != nil {
		return WorkspaceStatus{}, err
	}
	return ws.unstaged(cs), nil
}

func (ws *Workspace) unstaged(cs store.ChangeSet) WorkspaceStatus {
	afero.Walk(ws.fs, ws.dir, func(path string, info os.FileInfo, err error) error {

		return nil
	})
	return WorkspaceStatus{}
}

type WorkspaceStatus struct {
	Added    []Entry `json:"added,omitempty" yaml:"added,omitempty"`
	Deleted  []Entry `json:"deleted,omitempty" yaml:"deleted,omitempty"`
	Updated  []Entry `json:"updated,omitempty" yaml:"updated,omitempty"`
	Unstaged []Entry `json:"unstaged,omitempty" yaml:"unstaged,omitempty"`
}

type Entry struct {
	Object *cafs.Key      `json:"object,omitempty" yaml:"object,omitempty"`
	Path   string         `json:"path" yaml:"path"`
	Mtime  time.Time      `json:"mtime" yaml:"mtime"`
	Mode   store.FileMode `json:"mode" yaml:"mode"`
}

// ChangeSet captures the data for a change set in a bundle
type ChangeSet struct {
}
