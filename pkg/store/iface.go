package store

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

	List() ([]string, error)
	Get(string) (*Repo, error)
	Create(*Repo) error
	Update(*Repo) error
	Delete(string) error
	//Bundles(string) (BundleStore, error)
}

// A BundleStore manages persistence for bundle related data
type BundleStore interface {
	Store

	ListTopLevel() ([]Bundle, error)
	ListTopLevelIDs() ([]string, error)

	ListBranches() ([]string, error)
	HashForBranch(branch string) (string, error)
	CreateBranch(string, string) error
	DeleteBranch(string) error

	ListTags() ([]string, error)
	HashForTag(string) (string, error)
	CreateTag(string, string) error
	DeleteTag(string) error

	Create(message, branch, snapshot string, parents []string, changes ChangeSet) (string, bool, error)
	Get(string) (*Bundle, error)
	GetObject(string) (Entry, error)
	GetObjectForPath(string) (Entry, error)
	HashForPath(path string) (string, error)
}

// An StageMeta store manages the indices for file paths to
// hashes and the file info meta data
type StageMeta interface {
	Store

	Add(Entry) error
	Remove(string) error
	List() (ChangeSet, error)
	MarkDelete(*Entry) error
	Get(string) (Entry, error)
	HashFor(string) (string, error)
	Clear() error
}

// A SnapshotStore manages persistence for snapshot data
type SnapshotStore interface {
	Store

	Create(*Bundle) (*Snapshot, error)
	Get(string) (*Snapshot, error)
	GetForBundle(string) (*Snapshot, error)
}
