package model

import (
	"fmt"
	"strings"
	"time"
)

type LabelDescriptor struct {
	Name         string        `json:"name,omitempty" yaml:"name,omitempty"`
	BundleID     string        `json:"id" yaml:"id"`
	Timestamp    time.Time     `json:"timestamp,omitempty" yaml:"timestamp,omitempty"`
	Contributors []Contributor `json:"contributors" yaml:"contributors"`
	_            struct{}
}

// LabelDescriptors is a sortable slice of LabelDescriptor
type LabelDescriptors []LabelDescriptor

func (b LabelDescriptors) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}
func (b LabelDescriptors) Len() int {
	return len(b)
}
func (b LabelDescriptors) Less(i, j int) bool {
	return b[i].BundleID < b[j].BundleID
}
func (b LabelDescriptors) Last() LabelDescriptor {
	return b[len(b)-1]
}

func getArchivePathToLabels() string {
	return "labels/"
}

func GetArchivePathPrefixToLabels(repo string, prefixes ...string) string {
	return fmt.Sprint(getArchivePathToLabels(), repo+"/"+strings.Join(prefixes, "/"))
}

func GetArchivePathToLabel(repo string, labelName string) string {
	return fmt.Sprint(GetArchivePathPrefixToLabels(repo), labelName, ".json")
}
