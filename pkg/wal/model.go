package wal

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

// All the serializable model for the WAL.

type Entry struct {
	Token   string `json:"token" yaml:"token"`
	Payload string `json:"payload" yaml:"payload"`
}

const tokenGeneratorPath = "WALTokenGeneratorPath"
const maxConcurrency = 1024

func NewEntry(token string, payload string) *Entry {
	return &Entry{
		Token:   token,
		Payload: payload,
	}
}

func Unmarshal(b []byte) (*Entry, error) {
	if b == nil {
		return nil, fmt.Errorf("received nil entry to unmarshall")
	}
	var e Entry
	err := yaml.Unmarshal(b, &e)
	return &e, err
}

func Marshal(entry *Entry) ([]byte, error) {
	b, err := yaml.Marshal(entry)
	return b, err
}
