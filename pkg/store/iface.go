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

// An ObjectMeta store manages the indices for file paths to
// hashes and the file info meta data
type ObjectMeta interface {
	Initialize() error
	Close() error

	Add(Entry) error
	Remove(string) error
	List() ([]Entry, error)
	Get(string) (Entry, error)
	HashFor(string) (string, error)
	Clear() error
}
