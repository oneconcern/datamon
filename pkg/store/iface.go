package store

import "io"

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

// A BundleStore manages bundles within a repository
type BundleStore interface {
	//List() ([]Bundle, error)
	//Save(*Bundle) error
	//Delete(string) error
}

// A BlobStore is a content addressable file system
type BlobStore interface {
	Get(string) (io.Reader, error)
	Put(string, io.Reader) error
	Delete(string) error
	Keys() ([]string, error)
}
