package model

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	CurrentBundleVersion = 1
)

// BundleDescriptor represents a commit which is a file tree with the changes to the repository.
type BundleDescriptor struct {
	LeafSize               uint32        `json:"leafSize" yaml:"leafSize"` // Each bundles blobs are independently generated
	ID                     string        `json:"id" yaml:"id"`
	Message                string        `json:"message" yaml:"message"`
	Parents                []string      `json:"parents,omitempty" yaml:"parents,omitempty"`
	Timestamp              time.Time     `json:"timestamp,omitempty" yaml:"timestamp,omitempty"`
	Contributors           []Contributor `json:"contributors" yaml:"contributors"`
	BundleEntriesFileCount uint64        `json:"count" yaml:"count"`                         // Number of files which have BundleDescriptor Entries
	Version                uint64        `json:"version,omitempty" yaml:"version,omitempty"` // Version for the bundle
	_                      struct{}
}

// List of files part of a bundle.
type BundleEntries struct {
	BundleEntries []BundleEntry `json:"BundleEntries" yaml:"BundleEntries"`
	_             struct{}
}

type ArchivePathComponents struct {
	Repo            string
	BundleID        string
	ArchiveFileName string
}

// List of files, directories (empty) skipped
type BundleEntry struct {
	Hash         string      `json:"hash" yaml:"hash"`
	NameWithPath string      `json:"name" yaml:"name"`
	FileMode     os.FileMode `json:"mode" yaml:"mode"`
	Size         uint64      `json:"size" yaml:"size"`
	_            struct{}
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

func GetConsumablePathToBundle(bundleID string) string {
	return fmt.Sprint("./.datamon/", bundleID, ".json")
}

func GetConsumablePathToBundleFileList(bundleID string, index uint64) string {
	return fmt.Sprint("./.datamon/", bundleID, "-bundle-files-", index, ".json")
}

func GetArchivePathToBundle(repo string, bundleID string) string {
	return fmt.Sprint(getArchivePathToBundles(), repo, "/", bundleID, "/bundle.json")
}

func GetArchivePathPrefixToBundles(repo string) string {
	return fmt.Sprint(getArchivePathToBundles(), repo+"/")
}

func getArchivePathToBundles() string {
	return fmt.Sprint("bundles/")
}

func GetArchivePathToBundleFileList(repo string, bundleID string, index uint64) string {
	// <repo>-bundles/<bundle>/bundlefiles-<index>.json
	return fmt.Sprint(getArchivePathToBundles(), repo, "/", bundleID, "/bundle-files-", index, ".json")
}

func GetArchivePathComponents(archivePath string) (ArchivePathComponents, error) {
	cs := strings.SplitN(archivePath, "/", 4)
	return ArchivePathComponents{
		Repo:            cs[1],
		BundleID:        cs[2],
		ArchiveFileName: cs[3],
	}, nil // placeholder in case of mor parsing
}

func GetBundleTimeStamp() time.Time {
	t := time.Now()
	return t.UTC()
}

func IsGeneratedFile(file string) bool {
	// TODO: Need to find a way for AeroFs to convert to abs patch while honoring the fake root
	//path, err := filepath.Abs(file)
	match, _ := regexp.MatchString("^.datamon/*|^/.datamon/*|^/.datamon$|^.datamon$|^./.datamon/*|^./.datamon$", file)
	return match
}
