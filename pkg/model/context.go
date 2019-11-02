/*
 * Copyright © 2019 One Concern
 *
 */

package model

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

const ContextVersion = 1.0

// Context defines the details for a Datamon context.
type Context struct {
	Name      string `json:"name" yaml:"name"`           // Name for the context
	WAL       string `json:"wal" yaml:"wal"`             // WAL is the location for the log
	ReadLog   string `json:"readlog" yaml:"readlog"`     // Read log is the location for read log.
	Blob      string `json:"blob" yaml:"blob"`           // Blob is the location for the data blobs
	Metadata  string `json:"metadata" yaml:"metadata"`   // Metadata is the location for the immutable metadata
	VMetadata string `json:"vmetadata" yaml:"vmetadata"` // VMetadata is the location for the mutable versioned metadata.
	Version   uint64 `json:"version" yaml:"version"`     // Version for the
	_         struct{}
}

// GetPathToContext returns the path to the context descriptor.
func GetPathToContext(context string) string {
	return context + "/" + "context.yaml"
}

func GetWALName(context string, n string) string {
	if n != "" {
		return n
	}
	return context + "-wal"
}
func GetBlobName(context string, n string) string {
	if n != "" {
		return n
	}
	return context + "-blob"
}
func GetMetadataName(context string, n string) string {
	if n != "" {
		return n
	}
	return context + "-metadata"
}
func GetVMetadataName(context string, n string) string {
	if n != "" {
		return n
	}
	return context + "-v-metadata"
}
func GetReadLogName(context string, n string) string {
	if n != "" {
		return n
	}
	return context + "-read-log"
}
func UnmarshalContext(b []byte) (*Context, error) {
	if b == nil {
		return nil, fmt.Errorf("received nil entry to unmarshall")
	}
	var c Context
	err := yaml.Unmarshal(b, &c)
	return &c, err
}

func MarshalContext(c *Context) ([]byte, error) {
	b, err := yaml.Marshal(c)
	return b, err
}

func ValidateContext(context Context) error {
	var cause string
	switch {
	case context.Name == "":
		cause += "Name is empty. "
		fallthrough
	case context.ReadLog == "":
		cause += "ReadLog is empty. "
		fallthrough
	case context.Metadata == "":
		cause += "Metadata is empty. "
		fallthrough
	case context.VMetadata == "":
		cause += "VMetadata is empty. "
		fallthrough
	case context.Blob == "":
		cause += "Blob is empty. "
		fallthrough
	case context.WAL == "":
		cause += "WAL is empty. "
		fallthrough
	case context.Version > ContextVersion:
		cause += "Version higher than supported version"
	}
	if cause != "" {
		return fmt.Errorf("validation failed, cause = %s", cause)
	}
	return nil
}
