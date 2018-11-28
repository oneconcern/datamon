package model

import (
	"time"
)

// Snapshot represents the composed list of entries that are to be shown
type Snapshot struct {
	ID              string    `json:"id" yaml:"id"`
	Parents         []string  `json:"parents,omitempty" yaml:"parents,omitempty"`
	NewCommit       string    `json:"commit,omitempty" yaml:"commit,omitempty"`
	PreviousCommits []string  `json:"previous_commits,omitempty" yaml:"previous_commits,omitempty"`
	Entries         Entries   `json:"entries,omitempty" yaml:"entries,omitempty"`
	Timestamp       time.Time `json:"timestamp,omitempty" yaml:"timestamp,omitempty"`
	_               struct{}
}

// Snapshots represents a collection of snapshots
// this collection is ordered from most recent to oldest
type Snapshots []Snapshot

func (sn Snapshots) Len() int { return len(sn) }

func (sn Snapshots) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return sn[i].Timestamp.After(sn[j].Timestamp)
}

func (sn Snapshots) Swap(i, j int) {
	if j < 0 || i < 0 {
		return
	}
	sn[i], sn[j] = sn[j], sn[i]
}

// Push a snapshot into the priority queue
func (sn *Snapshots) Push(x interface{}) {
	item := x.(Snapshot)
	*sn = append(*sn, item)
}

// Pop the most recent snapshot from the queue
func (sn *Snapshots) Pop() interface{} {
	old := *sn
	n := len(old)
	if n == 0 {
		return nil
	}
	item := old[n-1]
	*sn = old[0 : n-1]
	return &item
}
