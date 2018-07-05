package store

import (
	"encoding/hex"
	"fmt"
	"os"
	"time"

	units "github.com/docker/go-units"
	blake2b "github.com/minio/blake2b-simd"
)

// Repo represents a repository in the trumpet
type Repo struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	TagsRef     map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
	BranchRef   map[string]string `json:"branches,omitempty" yaml:"branches,omitempty"`
	_           struct{}
}

// Bundle represents a commit which is a file tree with the changes to the repository.
type Bundle struct {
	ID         string        `json:"id" yaml:"id"`
	Message    string        `json:"message" yaml:"message"`
	Parents    []string      `json:"parents,omitempty" yaml:"parents,omitempty"`
	Changes    ChangeSet     `json:"changes" yaml:"changes"`
	Timestamp  time.Time     `json:"timestamp,omitempty" yaml:"timestamp,omitempty"`
	IsSnapshot bool          `json:"is_snapshot,omitempty" yaml:"is_snapshot,omitempty"`
	Committers []Contributor `json:"committers" yaml:"committers"`
	_          struct{}
}

// Contributor who created the object
type Contributor struct {
	Name  string `json:"name" yaml:"name"`
	Email string `json:"email" yaml:"email"`
	_     struct{}
}

func (c *Contributor) String() string {
	if c.Email == "" {
		return c.Name
	}
	if c.Name == "" {
		return c.Email
	}
	return fmt.Sprintf("%s <%s>", c.Name, c.Email)
}

// ChangeSet captures the data for a change set in a bundle
type ChangeSet struct {
	Added   []Entry `json:"added,omitempty" yaml:"added,omitempty"`
	Deleted []Entry `json:"deleted,omitempty" yaml:"deleted,omitempty"`
	Updated []Entry `json:"updated,omitempty" yaml:"updated,omitempty"`
	_       struct{}
}

// Hash the added files
func (cs *ChangeSet) Hash() (string, error) {
	// Compute hash of level 1 root key
	hasher, err := blake2b.New(&blake2b.Config{
		Size: blake2b.Size,
		Tree: &blake2b.Tree{
			Fanout:        0,
			MaxDepth:      2,
			LeafSize:      5 * units.MiB,
			NodeOffset:    0,
			NodeDepth:     1,
			InnerHashSize: blake2b.Size,
			IsLastNode:    true,
		},
	})
	if err != nil {
		return "", err
	}

	for _, v := range cs.Added {
		hasher.Write(UnsafeStringToBytes(v.Hash))
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// Entry for the stage or a bundle
type Entry struct {
	Path  string      `json:"path" yaml:"path"`
	Hash  string      `json:"hash" yaml:"hash"`
	Mtime time.Time   `json:"mtime" yaml:"mtime"`
	Mode  os.FileMode `json:"mode" yaml:"mode"`
	_     struct{}
}

// //go:generate jsonenums -type ObjectType

// // ObjectType describes the type of entry we're putting in the data store
// type ObjectType uint8

// const (
// 	//Unknown object type, this is normally an error or uninitalized field
// 	Unknown ObjectType = iota
// 	// File objects represent objects that have content
// 	File
// 	// Tree objects represent objects that contain files
// 	Tree
// 	// Commit objects are objects
// 	Commit
// )

// type Object struct {
// 	ID   string      `json:"id" yaml:"id"`
// 	Data interface{} `json:"data" yaml:"data"`
// 	Type ObjectType  `json:"type" yaml:"type"`
// 	_    struct{}
// }
