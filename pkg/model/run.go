/*
 * Copyright © 2019 One Concern
 *
 */

package model

import (
	"time"
)

type Run struct {
	Type        string      `json:"type" yaml:"type"` // Type of run
	Name        string      `json:"name" yaml:"name"` // Name for the run
	ID          string      `json:"id" yaml:"id"`     // Unique id for an instance of a run. Each time the same Type+Name is run it results in a new ID.
	Timestamp   time.Time   `json:"timestamp" yaml:"timestamp"`
	Contributor Contributor `json:"contributor" yaml:"contributor"`
	Steps       []Steps     `json:"steps" yaml:"steps"`
}

type FS struct {
	Context     string `json:"context" yaml:"context"`
	Repo        string `json:"repo" yaml:"repo"`
	BundleID    string `json:"bundleid" yaml:"bundleid"`
	Label       string `json:"label" yaml:"label"`
	Fuse        bool   `json:"fuse" yaml:"fuse"`
	Stream      bool   `json:"stream" yaml:"stream"`
	MountPath   string `json:"mountpath" yaml:"mountpath"`
	ReMountPath string `json:"remountpath" yaml:"remountpath"`
	ExportPath  string `json:"exportpath" yaml:"exportpath"`
	Name        string `json:"name" yaml:"name"`
}

type DB struct {
	Context  string `json:"context" yaml:"context"`
	Repo     string `json:"repo" yaml:"repo"`
	BundleID string `json:"bundleid" yaml:"bundleid"`
	Label    string `json:"label" yaml:"label"`
	Type     string `json:"type" yaml:"type"`
	Port     int    `json:"port" yaml:"port"`
}

type DataSetIn struct {
	Filesystems []FS `json:"filesystems" yaml:"filesystems"`
	Database    []DB `json:"database" yaml:"database"`
}

type DataSetOut struct {
	Filesystems []FS `json:"filesystems" yaml:"filesystems"`
	Database    []DB `json:"database" yaml:"database"`
}

type Steps struct {
	Containers []RunStepContainer `json:"containers" yaml:"containers"`
}

type RunStepContainer struct {
	ContainerID string `json:"containerid" yaml:"containerid"`
}

func GetPathToRun(id string) string {
	return id + "/run.yaml"
}
