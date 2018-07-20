package store

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/json-iterator/go"
)

// Repo represents a repository in the trumpet
type Repo struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	TagsRef     map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
	BranchRef   map[string]string `json:"branches,omitempty" yaml:"branches,omitempty"`
	_           struct{}          `json:"-" yaml:"-"`
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
	_          struct{}      `json:"-" yaml:"-"`
}

// Contributor who created the object
type Contributor struct {
	Name  string   `json:"name" yaml:"name"`
	Email string   `json:"email" yaml:"email"`
	_     struct{} `json:"-" yaml:"-"`
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
	Added   Entries  `json:"added,omitempty" yaml:"added,omitempty"`
	Deleted Entries  `json:"deleted,omitempty" yaml:"deleted,omitempty"`
	Updated Entries  `json:"updated,omitempty" yaml:"updated,omitempty"`
	_       struct{} `json:"-" yaml:"-"`
}

// Hash the added files
func (cs *ChangeSet) Hash() (string, error) {
	return cs.Added.Hash()
}

// FileMode type to wrap os.FileMode with a lossless json conversion
type FileMode os.FileMode

// MarshalJSON implements json.Marshaller
func (f FileMode) MarshalJSON() ([]byte, error) {
	return jsoniter.Marshal(strconv.FormatUint(uint64(uint32(f)), 8))
}

func (f FileMode) String() string {
	return strconv.FormatUint(uint64(uint32(f)), 8)
}

// UnmarshalJSON implements json.Unmarshaller
func (f *FileMode) UnmarshalJSON(data []byte) error {
	var str string
	if err := jsoniter.Unmarshal(data, &str); err != nil {
		return err
	}
	res, err := strconv.ParseUint(str, 8, 32)
	if err != nil {
		return err
	}
	*f = FileMode(uint32(res))
	return nil
}
