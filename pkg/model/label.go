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

func getArchivePathToLabels() string {
	return fmt.Sprint("labels/")
}

func GetArchivePathPrefixToLabels(repo string, prefixes ...string) string {
	return fmt.Sprint(getArchivePathToLabels(), repo+"/"+strings.Join(prefixes, "/"))
}

func GetArchivePathToLabel(repo string, labelName string) string {
	return fmt.Sprint(GetArchivePathPrefixToLabels(repo), labelName, ".json")
}
