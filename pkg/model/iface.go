package model

import "context"

type errorString string

func (e errorString) Error() string {
	return string(e)
}

const (
	// NameIsRequired error whenever a name is expected but not provided
	NameIsRequired errorString = "name is required"

	// IDIsRequired error whenever a name is expected but not provided
	IDIsRequired errorString = "id is required"

	// RepoAlreadyExists is returned when a repo is expected to not exist yet
	RepoAlreadyExists errorString = "repo already exists"

	// ObjectAlreadyExists is returned when a repo is expected to not exist yet
	ObjectAlreadyExists errorString = "object already exists"

	// RepoNotFound when a repository is not found
	RepoNotFound errorString = "repo not found"

	// ObjectNotFound when a repository is not found
	ObjectNotFound errorString = "object not found"

	// BundleNotFound when a bundle is not found
	BundleNotFound errorString = "bundle not found"

	// SnapshotNotFound when a bundle is not found
	SnapshotNotFound errorString = "snapshot not found"

	// BranchAlreadyExists is returned when a branch is expected to not exist yet
	BranchAlreadyExists errorString = "branch already exists"
)

// Store contains the common methods between all stores
type Store interface {
	Initialize() error
	Close() error
}

// A RepoStore manages repositories in a storage mechanism
type RepoStore interface {
	Store

	List(context.Context) ([]string, error)
	Get(context.Context, string) (*Repo, error)
	Create(context.Context, *Repo) error
	Update(context.Context, *Repo) error
	Delete(context.Context, string) error
}

// A BundleStore manages model for bundle related data
type BundleStore interface {
	Store

	ListTopLevel(context.Context) ([]Bundle, error)
	ListTopLevelIDs(context.Context) ([]string, error)

	ListBranches(context.Context) ([]string, error)
	HashForBranch(context.Context, string) (string, error)
	CreateBranch(context.Context, string, string) error
	DeleteBranch(context.Context, string) error

	ListTags(context.Context) ([]string, error)
	HashForTag(context.Context, string) (string, error)
	CreateTag(context.Context, string, string) error
	DeleteTag(context.Context, string) error

	Create(context.Context, string, string, string, []string, ChangeSet) (string, bool, error)
	Get(context.Context, string) (*Bundle, error)
	GetObject(context.Context, string) (Entry, error)
	GetObjectForPath(context.Context, string) (Entry, error)
	HashForPath(context.Context, string) (string, error)
}

// An StageMeta model manages the indices for file paths to
// hashes and the file info meta data
type StageMeta interface {
	Store

	Add(context.Context, Entry) error
	Remove(context.Context, string) error
	List(context.Context) (ChangeSet, error)
	MarkDelete(context.Context, *Entry) error
	Get(context.Context, string) (Entry, error)
	HashFor(context.Context, string) (string, error)
	Clear(context.Context) error
}

// A SnapshotStore manages model for snapshot data
type SnapshotStore interface {
	Store

	Create(context.Context, *Bundle) (*Snapshot, error)
	Get(context.Context, string) (*Snapshot, error)
	GetForBundle(context.Context, string) (*Snapshot, error)
}
