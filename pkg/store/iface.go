package store

type errorString string

func (e errorString) Error() string {
	return string(e)
}

const (
	// NameIsRequired error whenever a name is expected but not provided
	NameIsRequired errorString = "name is required"

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

	// BlobNotFound when a bundle is not found
	BlobNotFound errorString = "blob not found"
)

// A RepoStore manages repositories in a storage mechanism
type RepoStore interface {
	Initialize() error
	Close() error

	List() ([]string, error)
	Get(string) (*Repo, error)
	Create(*Repo) error
	Update(*Repo) error
	Delete(string) error
	//Bundles(string) (BundleStore, error)
}

type BundleStore interface {
	Initialize() error
	Close() error

	ListTopLevel() ([]Bundle, error)
	ListTopLevelIDs() ([]string, error)
	Create(message, branch, snapshot string, parents []string, changes ChangeSet) (string, bool, error)
	GetObject(string) (Entry, error)
	GetObjectForPath(string) (Entry, error)

	HashForPath(path string) (string, error)
}

// An StageMeta store manages the indices for file paths to
// hashes and the file info meta data
type StageMeta interface {
	Initialize() error
	Close() error

	Add(Entry) error
	Remove(string) error
	List() (ChangeSet, error)
	MarkDelete(*Entry) error
	Get(string) (Entry, error)
	HashFor(string) (string, error)
	Clear() error
}
