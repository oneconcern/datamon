// Package model describes the base objects manipulated by datamon.
//
// The package exposes a model for metadata.
//
// The object model for datamon is composed of:
//
//  Repos:
//    A datamon repository is analogous to a git repo. A repo is a dataset that has a unified lifecycle.
//    A particular version of the files in a repo is called a bundle.
//
//  Bundles:
//    A bundle is a point in time read-only view of a repo, composed of individual files.
//    This is analogous to a commit in git.
//
//  Labels:
//    A name given to a bundle, analogous to tags in git. Examples: Latest, production.
//
//  Contexts:
//    A context provides a way to define multiple instances of datamon. Example: development, production
//
//  WAL:
//    A write-ahead log tracks all changes to a repo.
package model
