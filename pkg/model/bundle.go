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
	// CurrentBundleVersion indicates the version of the bundle model
	CurrentBundleVersion = 1.0
)

// BundleDescriptor represents a commit which is a file tree with the changes to the repository.
type BundleDescriptor struct {
	LeafSize               uint32        `json:"leafSize" yaml:"leafSize"`                               // Bundles blobs are independently generated
	ID                     string        `json:"id" yaml:"id"`                                           // Unique ID for the bundle.
	Message                string        `json:"message" yaml:"message"`                                 // Message for the commit/bundle
	Parents                []string      `json:"parents,omitempty" yaml:"parents,omitempty"`             // Bundles with parent child relation
	Timestamp              time.Time     `json:"timestamp,omitempty" yaml:"timestamp,omitempty"`         // Local wall clock time
	Contributors           []Contributor `json:"contributors" yaml:"contributors"`                       // Contributor for the bundle
	BundleEntriesFileCount uint64        `json:"count" yaml:"count"`                                     // Number of files which have BundleDescriptor Entries
	Version                uint64        `json:"version,omitempty" yaml:"version,omitempty"`             // Version for the bundle
	Deduplication          string        `json:"deduplication,omitempty" yaml:"deduplication,omitempty"` // Type of deduplication used
	_                      struct{}
}

// BundleDescriptors is a sortable slice of BundleDescriptor
type BundleDescriptors []BundleDescriptor

func (b BundleDescriptors) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}
func (b BundleDescriptors) Len() int {
	return len(b)
}
func (b BundleDescriptors) Less(i, j int) bool {
	return b[i].ID < b[j].ID
}

// Last bundle descriptor in slice
func (b BundleDescriptors) Last() BundleDescriptor {
	return b[len(b)-1]
}

// BundleEntries are the files in this bundle.
type BundleEntries struct {
	BundleEntries []BundleEntry `json:"BundleEntries" yaml:"BundleEntries"`
	_             struct{}
}

// ArchivePathComponents defines the unique path parts to retrieve a file in a bundle
type ArchivePathComponents struct {
	Repo            string
	BundleID        string
	ArchiveFileName string
	LabelName       string
}

// BundleEntry describes a file in the bundle. Empty directories are skipped
type BundleEntry struct {
	Hash         string      `json:"hash" yaml:"hash"`
	NameWithPath string      `json:"name" yaml:"name"`
	FileMode     os.FileMode `json:"mode" yaml:"mode"`
	Size         uint64      `json:"size" yaml:"size"`
	_            struct{}
}

const (
	// ConsumableStorePathTypeDescriptor defines consumable store metadata of type "descriptor"
	ConsumableStorePathTypeDescriptor byte = iota
	// ConsumableStorePathTypeFileList defines consumable store metadata of type "file list"
	ConsumableStorePathTypeFileList
)

// ConsumableStorePathMetadata defines the metadata associated to a consumable store.
type ConsumableStorePathMetadata struct {
	Type     byte
	BundleID string
	Index    uint64
}

// GetConsumableStorePathMetadata is the inverse of GetConsumablePath* functions.
//
// The GetConsumablePath* functions return bundle.ConsumableStore keys (paths)
// given some parameters from the bundle.
//
// This function, given one of the paths returned by a GetConsumablePath* function,
// parses the path and returns the input values to that function.
func GetConsumableStorePathMetadata(path string) (ConsumableStorePathMetadata, error) {
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

// GetConsumablePathToBundle yields a path to some bundle metadata
func GetConsumablePathToBundle(bundleID string) string {
	path := fmt.Sprint(".datamon/", bundleID, ".yaml")
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

// GetConsumablePathToBundleFileList yields a path to some bundle file
func GetConsumablePathToBundleFileList(bundleID string, index uint64) string {
	path := fmt.Sprint(".datamon/", bundleID, "-bundle-files-", index, ".yaml")
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

// GetArchivePathToBundle yields a path in a repo to some bundle
func GetArchivePathToBundle(repo string, bundleID string) string {
	return fmt.Sprint(getArchivePathToBundles(), repo, "/", bundleID, "/bundle.yaml")
}

// GetArchivePathPrefixToBundles yields a path to all bundles in a repo
func GetArchivePathPrefixToBundles(repo string) string {
	return fmt.Sprint(getArchivePathToBundles(), repo+"/")
}

func getArchivePathToBundles() string {
	return fmt.Sprint("bundles/")
}

// GetArchivePathToBundleFileList yields a path to the file list of a bundle
func GetArchivePathToBundleFileList(repo string, bundleID string, index uint64) string {
	// <repo>-bundles/<bundle>/bundlefiles-<index>.yaml
	return fmt.Sprint(getArchivePathToBundles(), repo, "/", bundleID, "/bundle-files-", index, ".yaml")
}

var metaRe, flRe, genFileRe *regexp.Regexp

// GetArchivePathComponents yields all components from an archive path.
//
// NOTE: this function's design is converging on being able to return something meaningful
// given any path in the metadata archive, not just those corresponding to bundles.
//
// The return value might be changed to an interface type in later iterations.
func GetArchivePathComponents(archivePath string) (ArchivePathComponents, error) {
	cs := strings.SplitN(archivePath, "/", 4)
	if cs[0] == "labels" {
		if cs[3] != "label.yaml" {
			return ArchivePathComponents{}, fmt.Errorf("path is invalid, last element in the path should be label.yaml. components: %v, path: %s", cs, archivePath)
		}
		return ArchivePathComponents{
			LabelName: cs[2],
			Repo:      cs[1],
		}, nil
	}
	if cs[0] == "repos" {
		return ArchivePathComponents{Repo: cs[1]}, nil
		// TODO: implement more stringent checks. bundle_list_test.go needs refactoring
		//if cs[2] == "repo.yaml" {
		//}
		//return ArchivePathComponents{}, fmt.Errorf("path is invalid, last element in the path should be repo.yaml. components: %v, path: %s", cs, archivePath)
	}
	if cs[0] == "bundles" {
		return ArchivePathComponents{
			Repo:            cs[1],
			BundleID:        cs[2],
			ArchiveFileName: cs[3],
		}, nil // placeholder in case of mor parsing
		//if cs[3] == "bundle.yaml" {
		//}
		//return ArchivePathComponents{}, fmt.Errorf("path is invalid, last element in the path should be bundle.yaml. components: %v, path: %s", cs, archivePath)
	}
	return ArchivePathComponents{
		Repo:            cs[1],
		BundleID:        cs[2],
		ArchiveFileName: cs[3],
	}, nil // placeholder in case of mor parsing
	//return ArchivePathComponents{}, fmt.Errorf("path is invalid: %v, path: %s", cs, archivePath)
}

// GetBundleTimeStamp yields the current UTC time
func GetBundleTimeStamp() time.Time {
	t := time.Now()
	return t.UTC()
}

// IsGeneratedFile indicate if some file comes from auto-generation (e.g. .datamon files)
func IsGeneratedFile(file string) bool {
	// TODO: Need to find a way for AeroFs to convert to abs patch while honoring the fake root
	//path, err := filepath.Abs(file)
	return genFileRe.MatchString(file)
}

func init() {
	metaRe = regexp.MustCompile(`^\.datamon/(.*)\.yaml$`)
	flRe = regexp.MustCompile(`^(.*)-bundle-files-(.*)$`)
	genFileRe = regexp.MustCompile("^.datamon/*|^/.datamon/*|^/.datamon$|^.datamon$|^./.datamon/*|^./.datamon$")
}
