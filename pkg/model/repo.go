package model

import (
	"fmt"
	"time"
	"unicode"
)

// RepoDescriptor represents a commit which is a file tree with the changes to the repository.
type RepoDescriptor struct {
	Name        string      `json:"name,omitempty" yaml:"name,omitempty"`
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`
	Timestamp   time.Time   `json:"timestamp,omitempty" yaml:"timestamp,omitempty"`
	Contributor Contributor `json:"contributor,omitempty" yaml:"contributor,omitempty"`
}

// RepoDescriptors is a sortable slice of RepoDescriptor
type RepoDescriptors []RepoDescriptor

func (b RepoDescriptors) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}
func (b RepoDescriptors) Len() int {
	return len(b)
}
func (b RepoDescriptors) Less(i, j int) bool {
	return b[i].Name < b[j].Name
}

// Last returns the last entry in a slice of RepoDescriptors
func (b RepoDescriptors) Last() RepoDescriptor {
	return b[len(b)-1]
}

// GetArchivePathToRepoDescriptor returns the path for a repo descriptor
func GetArchivePathToRepoDescriptor(repo string) string {
	return fmt.Sprint("repos/", repo, "/", "repo.yaml")
}

func getArchivePathToRepos() string {
	return "repos/"
}

// GetArchivePathPrefixToRepos yields the path to all repos
func GetArchivePathPrefixToRepos() string {
	return fmt.Sprint(getArchivePathToRepos())
}

// ValidateRepo validates a repository descriptor: name and description are required, name only contains letters, digits or '-'.
func ValidateRepo(repo RepoDescriptor) error {
	if repo.Name == "" {
		return fmt.Errorf("empty field: repo name is empty")
	}
	if repo.Description == "" {
		return fmt.Errorf("empty field: repo description is empty")
	}
	for i, c := range repo.Name {
		if !unicode.IsDigit(c) && !unicode.IsLetter(c) && !unicode.Is(unicode.Hyphen, c) {
			return fmt.Errorf("invalid name: repo name:%s contains unsupported character \"%s\"",
				repo.Name,
				string([]rune(repo.Name)[i]))
		}
	}
	return nil
}
