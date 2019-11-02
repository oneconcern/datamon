package model

import (
	"fmt"
	"time"
)

// LabelDescriptor describes a label
type LabelDescriptor struct {
	Name         string        `json:"name,omitempty" yaml:"name,omitempty"`
	BundleID     string        `json:"id" yaml:"id"`
	Timestamp    time.Time     `json:"timestamp,omitempty" yaml:"timestamp,omitempty"`
	Contributors []Contributor `json:"contributors" yaml:"contributors"`
	_            struct{}
}

func getArchivePathToLabels() string {
	return fmt.Sprint("labels/")
}

// GetArchivePathPrefixToLabels gets the path to the labels.
func GetArchivePathPrefixToLabels(repo string) string {
	return fmt.Sprint(getArchivePathToLabels(), repo+"/")
}

// GetArchivePathPrefixToLabelPrefix gets the path to the label with a label prefix.
func GetArchivePathPrefixToLabelPrefix(repo string, prefix string) string {
	return fmt.Sprint(getArchivePathToLabels(), repo+"/"+prefix)
}

// GetArchivePathToLabel gets the path to the label descriptor.
func GetArchivePathToLabel(repo string, labelName string) string {
	return fmt.Sprint(GetArchivePathPrefixToLabels(repo), labelName, ".json")
}
