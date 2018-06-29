package store

import (
	"os"
	"time"
)

// Repo represents a repository in the trumpet
type Repo struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	TagsRef     map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
	BranchRef   map[string]string `json:"branches,omitempty" yaml:"branches,omitempty"`
}

// Bundle represents a commit which is a file tree with the changes to the repository.
type Bundle struct {
	ID      string   `json:"id" yaml:"id"`
	Parents []string `json:"parents" yaml:"parents"`
}

// Entry for the stage or a bundle
type Entry struct {
	Path  string      `json:"path" yaml:"path"`
	Hash  string      `json:"hash" yaml:"hash"`
	Mtime time.Time   `json:"mtime" yaml:"mtime"`
	Mode  os.FileMode `json:"mode" yaml:"mode"`
}
