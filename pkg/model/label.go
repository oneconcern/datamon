package model

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"net/mail"
)

// LabelDescriptor describes a label
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

// Last label in a LabelDescriptors slice
func (b LabelDescriptors) Last() LabelDescriptor {
	return b[len(b)-1]
}

func getArchivePathToLabels() string {
	return "labels/"
}

// GetArchivePathPrefixToLabels yields the path to labels in a repo, given some prefixes
func GetArchivePathPrefixToLabels(repo string, prefixes ...string) string {
	return fmt.Sprint(getArchivePathToLabels(), repo+"/"+strings.Join(prefixes, "/"))
}

// GetArchivePathToLabel gets the path to the label descriptor.
func GetArchivePathToLabel(repo string, labelName string) string {
	return fmt.Sprint(GetArchivePathPrefixToLabels(repo), labelName, "/label.yaml")
}

func ValidateLabel(label LabelDescriptor) error {
	if label.Name == "" {
		return fmt.Errorf("empty field: label name is empty")
	}
	if label.BundleID == "" {
		return fmt.Errorf("empty field: label bundleID is empty")
	}
	for i, c := range label.Name {
		// Note: useful reference https://www.compart.com/en/unicode/category
		if !unicode.IsDigit(c) &&
			!unicode.IsLetter(c) &&
			!unicode.Is(unicode.Hyphen, c) &&
			!unicode.Is(unicode.Pc, c) {
			return fmt.Errorf("invalid name: label name:%s contains unsupported character \"%s\"",
				label.Name,
				string([]rune(label.Name)[i]))
		}
	}
	for _, c := range label.Contributors {
		if c.Name == "" {
			return fmt.Errorf("name for contributor cannot be blank")
		}
		if c.Email == "" {
			return fmt.Errorf("email for contributor cannot be blank")
		}
		// TODO: Is this sufficient validation?
		_, err := mail.ParseAddress(c.Email)
		if err != nil {
			return fmt.Errorf("email validation failed for :%s err:%w", c.Email, err)
		}
	}
	return nil
}
