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

type Entry struct {
	Token   string `json:"token" yaml:"token"`
	Payload string `json:"payload" yaml:"payload"`
}

const TokenGeneratorPath = "WALTokenGeneratorPath"

func NewEntry(token string, payload string) *Entry {
	return &Entry{
		Token:   token,
		Payload: payload,
	}
}

func UnmarshalWAL(b []byte) (*Entry, error) {
	if b == nil {
		return nil, fmt.Errorf("received nil entry to unmarshall")
	}
	var e Entry
	err := yaml.Unmarshal(b, &e)
	return &e, err
}

func MarshalWAL(entry *Entry) ([]byte, error) {
	b, err := yaml.Marshal(entry)
	return b, err
}
