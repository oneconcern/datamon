package model

import (
	"fmt"
	"path"
	"time"

	"github.com/segmentio/ksuid"
)

// ConflictMode indicates the conflict detection mode defined for a diamond
type ConflictMode string

const (
	// IgnoreConflicts is the diamond mode with which conflicts are not handled (latest win, no track kept of clobbered files)
	IgnoreConflicts ConflictMode = "ignored"

	// EnableCheckpoints is the diamond mode with which conflicts are explicitly handled and saved as "checkpoints" (incremental upload)
	EnableCheckpoints ConflictMode = "enable-checkpoints"

	// EnableConflicts is the diamond mode with which conflicts are detected and saved as "conflicts"
	EnableConflicts ConflictMode = "enable-conflicts"

	// ForbidConflicts is the diamond mode with which conflicts result in failure to commit
	ForbidConflicts ConflictMode = "forbids-conflicts"
)

// DiamondState models the running status of an ungoing diamond workflow
type DiamondState string

const (
	// DiamondInitialized is the state of an initialized diamond
	DiamondInitialized DiamondState = "initialized"

	// DiamondDone indicates the diamond has completed with a successful commit. This is a terminal state.
	DiamondDone DiamondState = "done"

	// DiamondCanceled indicates the diamond has completed with a cancel. This is a terminal state.
	DiamondCanceled DiamondState = "canceled"
)

const (
	done                         = "done"
	running                      = "running"
	ext                          = ".yaml"
	diamondPrefix                = "diamond-"
	splitPrefix                  = "split-"
	diamondFinalDescriptorFile   = diamondPrefix + done + ext
	splitFinalDescriptorFile     = splitPrefix + done + ext
	diamondInitialDescriptorFile = diamondPrefix + running + ext
	splitInitialDescriptorFile   = splitPrefix + running + ext
)

// IsValid checks the value of a diamond state
func (s DiamondState) IsValid() bool {
	switch s {
	case DiamondInitialized, DiamondDone, DiamondCanceled:
		return true
	default:
		return false
	}
}

func (s DiamondState) String() string {
	return string(s)
}

// SplitState models the running status of an ungoing diamond split
type SplitState string

const (
	// SplitDone is the state of a completed split. This is a terminal state.
	SplitDone SplitState = done

	// SplitRunning is the state of running split
	SplitRunning SplitState = running
)

// IsValid checks the value of a diamond state
func (s SplitState) IsValid() bool {
	switch s {
	case SplitDone, SplitRunning:
		return true
	default:
		return false
	}
}

func (s SplitState) String() string {
	return string(s)
}

// DiamondDescriptor models a diamond's metadata
type DiamondDescriptor struct {
	DiamondID      string            `json:"diamondID" yaml:"diamondID"`
	StartTime      time.Time         `json:"startTime" yaml:"startTime"`                           // documentary: when the diamond was initialized
	EndTime        time.Time         `json:"endTime,omitempty" yaml:"endTime,omitempty"`           // documentary: the diamond completion time (i.e. when the diamond reached a terminal state)
	State          DiamondState      `json:"state" yaml:"state"`                                   // the captured state of the diamond
	Mode           ConflictMode      `json:"mode" yaml:"mode"`                                     // the conflict handling mode defined at commit time. The default value is "enable-conflicts"
	HasConflicts   bool              `json:"hasConflicts,omitempty" yaml:"hasConflicts,omitempty"` // documentary snapshot of the outcome of conflict handling after commit
	HasCheckpoints bool              `json:"hasCheckpoints,omitempty" yaml:"hasCheckpoints,omitempty"`
	Tag            string            `json:"tag,omitempty" yaml:"tag,omitempty"`           // user-defined tag to taint logs
	BundleID       string            `json:"bundleID,omitempty" yaml:"bundleID,omitempty"` // keeps track of the produced bundle, when the diamond is successfully committed
	Splits         []SplitDescriptor `json:"splits,omitempty" yaml:"splits,omitempty"`     // documentary snapshot of the splits collected after a successful commit
	_              struct{}
}

// DiamondDescriptors is a sortable slice of DiamondDescriptor
type DiamondDescriptors []DiamondDescriptor

func (b DiamondDescriptors) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}
func (b DiamondDescriptors) Len() int {
	return len(b)
}
func (b DiamondDescriptors) Less(i, j int) bool {
	if !b[i].StartTime.IsZero() && !b[j].StartTime.IsZero() {
		return b[i].StartTime.Before(b[j].StartTime)
	}
	return b[i].DiamondID < b[j].DiamondID
}

// Last bundle descriptor in slice
func (b DiamondDescriptors) Last() DiamondDescriptor {
	return b[len(b)-1]
}

func defaultDiamondDescriptor() *DiamondDescriptor {
	diamondID, err := ksuid.NewRandom()
	if err != nil {
		panic(fmt.Sprintf("cannot generate random ksuid: %v", err))
	}
	return &DiamondDescriptor{
		DiamondID: diamondID.String(),
		StartTime: GetBundleTimeStamp(),
		State:     DiamondInitialized,
		Mode:      EnableConflicts,
	}
}

// NewDiamondDescriptor builds a new DiamondDescriptor
func NewDiamondDescriptor(opts ...DiamondDescriptorOption) *DiamondDescriptor {
	d := defaultDiamondDescriptor()
	for _, apply := range opts {
		apply(d)
	}
	return d
}

// SplitDescriptor models the metadata about a given split within a diamond
type SplitDescriptor struct {
	SplitID               string        `json:"splitID" yaml:"splitID"`
	StartTime             time.Time     `json:"startTime" yaml:"startTime"`                 // documentary: when the split was started
	EndTime               time.Time     `json:"endTime,omitempty" yaml:"endTime,omitempty"` // documentary: the split completion time (i.e. when the split has reached a terminal state)
	State                 SplitState    `json:"state" yaml:"state"`                         // the state of this split
	Contributors          []Contributor `json:"contributors" yaml:"contributors"`           // contributors to include in the resulting bundle
	GenerationID          string        `json:"generationID" yaml:"generationID"`           // unique location of index files used in final state (other possibly written locations are ignored)
	SplitEntriesFileCount uint64        `json:"count" yaml:"count"`                         // number of index files in this split
	Tag                   string        `json:"tag,omitempty" yaml:"tag,omitempty"`         // user-defined tag to taint logs

	_ struct{}
}

// SplitDescriptors is a sortable slice of SplitDescriptor
type SplitDescriptors []SplitDescriptor

func (b SplitDescriptors) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}
func (b SplitDescriptors) Len() int {
	return len(b)
}
func (b SplitDescriptors) Less(i, j int) bool {
	if !b[i].StartTime.IsZero() && !b[j].StartTime.IsZero() {
		return b[i].StartTime.Before(b[j].StartTime)
	}
	return b[i].SplitID < b[j].SplitID
}

// Last bundle descriptor in slice
func (b SplitDescriptors) Last() SplitDescriptor {
	return b[len(b)-1]
}

func defaultSplitDescriptor() *SplitDescriptor {
	splitID, err := ksuid.NewRandom()
	if err != nil {
		panic(fmt.Sprintf("cannot generate random ksuid: %v", err))
	}
	return &SplitDescriptor{
		SplitID:      splitID.String(),
		StartTime:    GetBundleTimeStamp(),
		State:        SplitRunning,
		Contributors: make([]Contributor, 0, 1),
	}
}

// NewSplitDescriptor builds a new SplitDescriptor
func NewSplitDescriptor(opts ...SplitDescriptorOption) *SplitDescriptor {
	s := defaultSplitDescriptor()
	for _, apply := range opts {
		apply(s)
	}
	return s
}

func getArchivePathToDiamonds() string {
	return "diamonds/"
}

// GetArchivePathPrefixToDiamonds yields a path to all diamonds in a repo.
//
// Example:
//   diamonds/{repo}/
func GetArchivePathPrefixToDiamonds(repo string) string {
	return fmt.Sprint(getArchivePathToDiamonds(), repo+"/")
}

// GetArchivePathToDiamond yields a path in a repo to the descriptor of a diamond in any state.
//
// Example:
//   diamonds/{repo}/{diamond}/diamond-*.yaml
func GetArchivePathToDiamond(repo, diamondID string, state DiamondState) string {
	var suffix string
	switch state {
	case DiamondInitialized:
		suffix = running
	default:
		suffix = done
	}
	return fmt.Sprint(getArchivePathToDiamonds(), repo, "/", diamondID, "/", diamondPrefix+suffix+ext)
}

// GetArchivePathToFinalDiamond yields a path in a repo to the descriptor of a diamond in a final state.
//
// Example:
//   diamonds/{repo}/{diamond}/diamond-done.yaml
func GetArchivePathToFinalDiamond(repo, diamondID string) string {
	return fmt.Sprint(getArchivePathToDiamonds(), repo, "/", diamondID, "/", diamondFinalDescriptorFile)
}

// GetArchivePathToInitialDiamond yields a path in a repo to the descriptor of a diamond in an initialized state.
//
// Example:
//   diamonds/{repo}/{diamond}/diamond-running.yaml
func GetArchivePathToInitialDiamond(repo, diamondID string) string {
	return fmt.Sprint(getArchivePathToDiamonds(), repo, "/", diamondID, "/", diamondInitialDescriptorFile)
}

// GetArchivePathPrefixToSplits yields a path to all splits in a diamond in a repo.
//
// Example:
//   diamonds/{repo}/{diamond}/
func GetArchivePathPrefixToSplits(repo, diamondID string) string {
	return fmt.Sprint(getArchivePathToDiamonds(), repo, "/", diamondID, "/", "splits", "/")
}

// GetArchivePathToSplit yields a path in a repo to the descriptor of a split in any state.
//
// Example:
//   diamonds/{repo}/{diamond}/splits/{split}/split-*.yaml
func GetArchivePathToSplit(repo, diamondID, splitID string, state SplitState) string {
	var suffix string
	switch state {
	case SplitRunning:
		suffix = running
	default:
		suffix = done
	}
	return fmt.Sprint(getArchivePathToDiamonds(), repo, "/", diamondID, "/", "splits", "/", splitID, "/", splitPrefix+suffix+ext)
}

// GetArchivePathToFinalSplit yields a path in a repo to the descriptor of a split in a final state.
//
// Example:
//   diamonds/{repo}/{diamond}/splits/{split}/split-done.yaml
func GetArchivePathToFinalSplit(repo, diamondID, splitID string) string {
	return fmt.Sprint(getArchivePathToDiamonds(), repo, "/", diamondID, "/", "splits", "/", splitID, "/", splitFinalDescriptorFile)
}

// GetArchivePathToInitialSplit yields a path in a repo to the descriptor of a split in a running state.
//
// Example:
//   diamonds/{repo}/{diamond}/splits/split}/split-running.yaml
func GetArchivePathToInitialSplit(repo, diamondID, splitID string) string {
	return fmt.Sprint(getArchivePathToDiamonds(), repo, "/", diamondID, "/", "splits", "/", splitID, "/", splitInitialDescriptorFile)
}

// GetArchivePathToSplitFileList yields a path to the list of the files in a diamond split
//
// Example:
//   diamonds/{repo}/{diamond}/splits/{split}/{generation}/bundle-files-{index}.yaml
func GetArchivePathToSplitFileList(repo, diamondID, splitID string, generationID string, index uint64) string {
	return fmt.Sprint(getArchivePathToDiamonds(), repo, "/", diamondID, "/", "splits", "/", splitID, "/", generationID, "/", splitFilesIndexPrefix, index, ext)
}

// GenerateConflictPath builds a path in the dataset to save conflicting files
func GenerateConflictPath(splitID, pth string) string {
	return path.Join(".conflicts", splitID, pth)
}

// GenerateCheckpointPath builds a path in the dataset to save checkpointed files
func GenerateCheckpointPath(splitID, pth string) string {
	return path.Join(".checkpoints", splitID, pth)
}
