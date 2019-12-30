package model

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	// descriptor files (object metadata)

	repoDescriptorFile     = "repo.yaml"
	labelDescriptorFile    = "label.yaml"
	bundleDescriptorFile   = "bundle.yaml"
	contextDescriptorFile  = "context.yaml"
	bundleFilesIndexPrefix = "bundle-files-"
)

var isBundleFileIndexRe *regexp.Regexp

func init() {
	isBundleFileIndexRe = regexp.MustCompile(`^` + bundleFilesIndexPrefix + `(\d+)\.yaml$`)
}

// ArchivePathComponents defines the unique path parts to retrieve a file in a bundle
type ArchivePathComponents struct {
	Repo            string
	BundleID        string
	ArchiveFileName string
	LabelName       string
	Context         string
}

// GetArchivePathComponents yields all components from an archive path.
//
// NOTE: this function's design is converging on being able to return something meaningful
// given any path in the metadata archive, not just those corresponding to bundles.
//
// The return value might be changed to an interface type in later iterations.
func GetArchivePathComponents(archivePath string) (ArchivePathComponents, error) {
	const (
		maxPos     = 4
		labelPos   = 3 // as in: labels/{repo}/{label}/label.yaml
		repoPos    = 2 // as in: repos/{repo}/repo.yaml
		bundlePos  = 3 // as in: bundles/{repo}/{bundleID}/bundle.yaml
		contextPos = 2 // as in: contexts/{context}/context.yaml
	)
	cs := strings.SplitN(archivePath, "/", maxPos)
	switch cs[0] { // we always have at least 1 element
	case "labels":
		if len(cs) < labelPos+1 {
			return ArchivePathComponents{},
				fmt.Errorf("path is invalid: expect path to label to have %d parts: %s", labelPos+1, archivePath)
		}
		if cs[labelPos] != labelDescriptorFile {
			return ArchivePathComponents{},
				fmt.Errorf("path is invalid, last element in the path should be %q. components: %v, path: %s",
					labelDescriptorFile, cs, archivePath)
		}
		return ArchivePathComponents{
			ArchiveFileName: cs[labelPos],
			LabelName:       cs[labelPos-1],
			Repo:            cs[labelPos-2],
		}, nil
	case "repos":
		if len(cs) < repoPos+1 {
			return ArchivePathComponents{},
				fmt.Errorf("path is invalid: expect path to repo to have %d parts: %s", labelPos+1, archivePath)
		}
		if cs[repoPos] != repoDescriptorFile {
			return ArchivePathComponents{},
				fmt.Errorf("path is invalid, last element in the path should be %q. components: %v, path: %s",
					repoDescriptorFile, cs, archivePath)
		}
		return ArchivePathComponents{
			ArchiveFileName: cs[repoPos],
			Repo:            cs[repoPos-1],
		}, nil
	case "bundles":
		if len(cs) < bundlePos+1 {
			return ArchivePathComponents{},
				fmt.Errorf("path is invalid: expect path to bundle to have %d parts: %s", bundlePos+1, archivePath)
		}
		switch {
		case cs[bundlePos] == "": // empty bundle
			fallthrough
		case cs[bundlePos] == bundleDescriptorFile:
			fallthrough
		case isBundleFileIndexRe.MatchString(cs[bundlePos]):
			return ArchivePathComponents{
				ArchiveFileName: cs[bundlePos],
				BundleID:        cs[bundlePos-1],
				Repo:            cs[bundlePos-2],
			}, nil // placeholder in case of more parsing
		default:
			return ArchivePathComponents{},
				fmt.Errorf("path is invalid, last element in the path should be either empty, %q or \"%s[nnn].yaml\". components: %v, path: %s",
					bundleDescriptorFile, bundleFilesIndexPrefix, cs, archivePath)
		}
	case "contexts":
		if len(cs) < contextPos+1 {
			return ArchivePathComponents{},
				fmt.Errorf("path is invalid: expect path to context to have %d parts: %s", contextPos+1, archivePath)
		}
		return ArchivePathComponents{
			ArchiveFileName: cs[contextPos],
			Context:         cs[contextPos-1],
		}, nil // placeholder in case of more parsing
	default:
		return ArchivePathComponents{}, fmt.Errorf("path is invalid: %v, path: %s", cs, archivePath)
	}
}
