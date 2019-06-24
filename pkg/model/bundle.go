package model

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
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
	LabelName       string
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

const (
	ConsumableStorePathTypeDescriptor = iota
	ConsumableStorePathTypeFileList
)

type ConsumableStorePathMetadata struct {
	Type     byte
	BundleID string
	Index    uint64
}

type ConsumableStorePathMetadataErr struct {
	msg string
}

func (e ConsumableStorePathMetadataErr) Error() string {
	return e.msg
}

/**
 * this function is the inverse of GetConsumablePath* functions.
 * the GetConsumablePath* functions return bundle.ConsumableStore keys (paths)
 * given some parameters from the bundle.
 * this function, given one of the paths returned by a GetConsumablePath* function,
 * parses the path and returns the input values to that function.
 */
func GetConsumableStorePathMetadata(path string) (ConsumableStorePathMetadata, error) {
	metaRe := regexp.MustCompile(`^\.datamon/(.*)\.json$`)
	flRe := regexp.MustCompile(`^(.*)-bundle-files-(.*)$`)
	info := ConsumableStorePathMetadata{}
	metaMatch := metaRe.FindStringSubmatch(path)
	if metaMatch == nil {
		return ConsumableStorePathMetadata{},
			ConsumableStorePathMetadataErr{msg: fmt.Sprintf("not a metadata path '%v'", path)}
	}
	metaName := metaMatch[1]
	flMatch := flRe.FindStringSubmatch(metaName)
	if flMatch == nil {
		info.Type = ConsumableStorePathTypeDescriptor
		info.BundleID = metaName
	} else {
		info.Type = ConsumableStorePathTypeFileList
		info.BundleID = flMatch[1]
		index, err := strconv.Atoi(flMatch[2])
		if err != nil {
			return ConsumableStorePathMetadata{}, err
		}
		info.Index = uint64(index)
	}
	return info, nil
}

func GetConsumablePathToBundle(bundleID string) string {
	path := fmt.Sprint(".datamon/", bundleID, ".json")
	info, err := GetConsumableStorePathMetadata(path)
	if err != nil {
		panic(fmt.Errorf("path not valid against inverse function (programming error): %v", err))
	}
	if info.Type != ConsumableStorePathTypeDescriptor {
		panic(fmt.Errorf("unexpected type from inverse function (programming error)"))
	}
	if info.BundleID != bundleID {
		panic(fmt.Errorf("inverse function bundle id '%v' does not match provided id '%v' (programming error)",
			info.BundleID, bundleID))
	}
	return path
}

func GetConsumablePathToBundleFileList(bundleID string, index uint64) string {
	path := fmt.Sprint(".datamon/", bundleID, "-bundle-files-", index, ".json")
	info, err := GetConsumableStorePathMetadata(path)
	if err != nil {
		panic(fmt.Errorf("path not valid against inverse function (programming error): %v", err))
	}
	if info.Type != ConsumableStorePathTypeFileList {
		panic(fmt.Errorf("unexpected type from inverse function (programming error)"))
	}
	if info.BundleID != bundleID {
		panic(fmt.Errorf("inverse function bundle id '%v' does not match provided id '%v' (programming error)",
			info.BundleID, bundleID))
	}
	if info.Index != index {
		panic(fmt.Errorf("inverse function index '%v' does not match provided index '%v' (programming error)",
			info.Index, index))
	}
	return path
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

var labelNameRe *regexp.Regexp

/* this function's design is converging on being able to return something meaningful
 * given any path in the metadata archive, not just those corresponding to bundles.
 *
 * the return value might be changed to an interface type in later iterations.
 */
func GetArchivePathComponents(archivePath string) (ArchivePathComponents, error) {
	cs := strings.SplitN(archivePath, "/", 4)
	if cs[0] == "labels" {
		labelBasenameMatches := labelNameRe.FindStringSubmatch(cs[2])
		if labelBasenameMatches == nil {
			return ArchivePathComponents{}, fmt.Errorf(
				"expected label basename %s to match regexp %s",
				cs[2], labelNameRe.String())
		}
		labelName := labelBasenameMatches[1]

		return ArchivePathComponents{
			LabelName: labelName,
			Repo:      cs[1],
		}, nil
	}
	if cs[2] == "repo.json" {
		return ArchivePathComponents{Repo: cs[1]}, nil
	}
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

func init() {
	labelNameRe = regexp.MustCompile(`^(.*)\.json$`)
}
