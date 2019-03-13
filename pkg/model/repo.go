package model

import (
	"fmt"
	"time"
	"unicode"
)

// BundleDescriptor represents a commit which is a file tree with the changes to the repository.
type RepoDescriptor struct {
	Name        string      `json:"name,omitempty" yaml:"name,omitempty"`
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`
	Timestamp   time.Time   `json:"timestamp,omitempty" yaml:"timestamp,omitempty"`
	Contributor Contributor `json:"contributor,omitempty" yaml:"contributor,omitempty"`
}

func GetArchivePathToRepoDescriptor(repo string) string {
	return fmt.Sprint("repos/", repo, "/", "repo.json")
}

func getArchivePathToRepos() string {
	return fmt.Sprint("repos/")
}

func GetArchivePathPrefixToRepos() string {
	return fmt.Sprint(getArchivePathToRepos())
}

func Validate(repo RepoDescriptor) error {
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
