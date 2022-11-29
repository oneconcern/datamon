package model

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/segmentio/ksuid"
)

const (
	// descriptor files (object metadata)
	repoDescriptorFile    = "repo.yaml"
	labelDescriptorFile   = "label.yaml"
	bundleDescriptorFile  = "bundle.yaml"
	contextDescriptorFile = "context.yaml"
	reverseIndexFile      = "reverse-lookup.yaml"
	purgeLockFile         = "purge.lock"

	// file index files
	bundleFilesIndexPrefix = "bundle-files-"
	splitFilesIndexPrefix  = bundleFilesIndexPrefix
)

var isBundleFileIndexRe, isSplitIndexFileRe *regexp.Regexp

func init() {
	isBundleFileIndexRe = regexp.MustCompile(`^` + bundleFilesIndexPrefix + `(\d+)\.yaml$`)
	isSplitIndexFileRe = isBundleFileIndexRe
}

// ArchivePathComponents defines the unique path parts to retrieve a file in a bundle
type ArchivePathComponents struct {
	Repo            string
	BundleID        string
	ArchiveFileName string
	LabelName       string
	Context         string
	DiamondID       string
	SplitID         string
	GenerationID    string
	IsFinalState    bool
}

// GetArchivePathComponents yields all metadata components from a parsed archive path.
func GetArchivePathComponents(archivePath string) (ArchivePathComponents, error) {
	const (
		maxPos     = 7
		labelPos   = 3 // as in: labels/{repo}/{label}/label.yaml
		repoPos    = 2 // as in: repos/{repo}/repo.yaml
		bundlePos  = 3 // as in: bundles/{repo}/{bundleID}/bundle.yaml
		contextPos = 2 // as in: contexts/{context}/context.yaml
		diamondPos = 3 // as in: diamonds/{repo}/{diamond-id}/diamond-{running|done}.yaml
		splitPos   = 5 // as in: diamonds/{repo}/{diamond-id}/splits/{split-id}/...
		indexPos   = 6 // as in: diamonds/{repo}/{diamond-id}/splits/{split-id}/{generation-id}/...
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

	case "diamonds":
		if len(cs) < diamondPos+1 {
			return ArchivePathComponents{},
				fmt.Errorf("path is invalid: expect path to diamond to have %d parts: %s", diamondPos+1, archivePath)
		}

		diamondID := cs[diamondPos-1]
		if _, err := ksuid.Parse(diamondID); err != nil {
			return ArchivePathComponents{},
				fmt.Errorf("expected {diamond-id} %q to be a ksuid: %s", diamondID, archivePath)
		}

		switch {
		case cs[diamondPos] == "": // empty diamond
			fallthrough
		case cs[diamondPos] == diamondInitialDescriptorFile || cs[diamondPos] == diamondFinalDescriptorFile:
			return ArchivePathComponents{
				ArchiveFileName: cs[diamondPos],
				Repo:            cs[1],
				DiamondID:       diamondID,
				IsFinalState:    cs[diamondPos] == diamondFinalDescriptorFile,
			}, nil
		default:
			if len(cs) < splitPos {
				return ArchivePathComponents{},
					fmt.Errorf("path is invalid: expect path to split to have %d parts: %s", splitPos, archivePath)
			}
			splitID := cs[splitPos-1]
			if splitID == "" {
				return ArchivePathComponents{
					ArchiveFileName: "",
					Repo:            cs[1],
					DiamondID:       diamondID,
				}, nil
			}
			// NOTE: the splitID may not be a KSUID

			if len(cs) < splitPos+1 {
				return ArchivePathComponents{},
					fmt.Errorf("path is invalid: expect path to split to have %d parts: %s", splitPos+1, archivePath)
			}

			switch {
			case cs[splitPos] == "":
				fallthrough
			case cs[splitPos] == splitInitialDescriptorFile || cs[splitPos] == splitFinalDescriptorFile:
				return ArchivePathComponents{
					ArchiveFileName: cs[splitPos],
					Repo:            cs[1],
					DiamondID:       diamondID,
					SplitID:         splitID,
					IsFinalState:    cs[splitPos] == splitFinalDescriptorFile,
				}, nil
			case len(cs) > indexPos:
				generationID := cs[indexPos-1]
				if _, err := ksuid.Parse(generationID); err != nil {
					return ArchivePathComponents{},
						fmt.Errorf("expected {generation-id} %q to be a ksuid: %s", generationID, archivePath)
				}

				if len(cs) > indexPos+1 || !isSplitIndexFileRe.MatchString(cs[indexPos]) {
					return ArchivePathComponents{},
						fmt.Errorf("path is invalid, last element in the path should be \"%s[nnn].yaml\". components: %v, path: %s",
							splitFilesIndexPrefix, cs, archivePath)
				}

				return ArchivePathComponents{
					ArchiveFileName: cs[indexPos],
					Repo:            cs[1],
					DiamondID:       diamondID,
					SplitID:         splitID,
					GenerationID:    generationID,
					IsFinalState:    cs[splitPos] == splitFinalDescriptorFile,
				}, nil

			default:
				return ArchivePathComponents{},
					fmt.Errorf("path is invalid, last element in the path should be either %q or %q. components: %v, path: %s",
						splitInitialDescriptorFile, splitFinalDescriptorFile, cs, archivePath)
			}
		}
	default:
		return ArchivePathComponents{}, fmt.Errorf("path is invalid: %v, path: %s", cs, archivePath)
	}
}

func PurgeLock() string {
	return purgeLockFile
}

func ReverseIndex() string {
	return reverseIndexFile
}
