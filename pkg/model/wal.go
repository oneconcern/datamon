/*
 * Copyright © 2019 One Concern
 *
 */

package model

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

// All the serializable model for the WAL.

// Entry defines a Write Ahead Log entry
type Entry struct {
	Token   string `json:"token" yaml:"token"`
	Payload string `json:"payload" yaml:"payload"`
}

// TokenGeneratorPath is the path to the WAL token generator
const TokenGeneratorPath = "WALTokenGeneratorPath"

// NewEntry creates a new entry for the WAL
func NewEntry(token string, payload string) *Entry {
	return &Entry{
		Token:   token,
		Payload: payload,
	}
}

// UnmarshalWAL unmarshals a WAL entry from a YAML descriptor
func UnmarshalWAL(b []byte) (*Entry, error) {
	if b == nil {
		return nil, fmt.Errorf("received nil entry to unmarshall")
	}
	var e Entry
	err := yaml.Unmarshal(b, &e)
	return &e, err
}

// MarshalWAL marshals a WAL entry as a YAML descriptor
func MarshalWAL(entry *Entry) ([]byte, error) {
	b, err := yaml.Marshal(entry)
	return b, err
}
