package model

import (
	"github.com/json-iterator/go"

	"os"
	"strconv"
)

// Repo represents a repository in the datamon network
type Repo struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	TagsRef     map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
	BranchRef   map[string]string `json:"branches,omitempty" yaml:"branches,omitempty"`
	_           struct{}
}

// ChangeSet captures the data for a change set in a bundle
type ChangeSet struct {
	Added   Entries `json:"added,omitempty" yaml:"added,omitempty"`
	Deleted Entries `json:"deleted,omitempty" yaml:"deleted,omitempty"`
	Updated Entries `json:"updated,omitempty" yaml:"updated,omitempty"`
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
