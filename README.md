[![CircleCI](https://circleci.com/gh/oneconcern/datamon/tree/master.svg?style=svg&circle-token=e827ee1509892d8ba85db2a819b692ca451a7a97)](https://circleci.com/gh/oneconcern/datamon/tree/master)
[![GitHub release](https://img.shields.io/github/v/release/oneconcern/datamon)](https://github.com/oneconcern/datamon/releases/latest)
[![license](https://img.shields.io/badge/license-MIT-green)](https://raw.githubusercontent.com/oneconcern/datamon/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/oneconcern/datamon?status.svg)](http://godoc.org/github.com/oneconcern/datamon)

# Datamon

Datamon is a data science tool sponsored by [OneConcern](https://www.oneconcern.com) that helps managing data at scale.

## Primer

### Goals

The primary goal of datamon is to manage versioned data at rest, providing CLI tools for creation, access and tracking
in an environment where data repositories and their lifecycles are linked.

Datamon links the various sources of data, how they are processed and tracks the
output/new data that is generated from the existing data.

[More on design and architecture](docs/design.md).

### Environment

Although flexible in its concepts and architecture, the current version of datamon is primarily developed and tested 
against the Google Cloud environment. Note that AWS S3 storage buckets are supported (see [datamover tool](docs/datamover.md)).

#### Storage backends

Datamon supports the following cloud storage backends:
* Google Cloud Storage 
* AWS S3

### [Concepts](docs/concepts.md)

- [**Repo**](docs/concepts.md#repo): analogous to a git repo. A repo in datamon is a dataset that has a unified lifecycle.
- [**Bundle**](docs/concepts.md#bundle): a bundle is a point in time read-only view of a rep:branch and is composed of individual files. Analogous to a commit in git.
- [**Label**](docs/concepts.md#label): a name given to a bundle, analogous to tags in git. Examples: Latest, production.
- [**Context**](docs/concepts.md#context): a context provides a way to define multiple instances of datamon.
- [**Write Ahead Log**](docs/concepts.md#write-ahead-log): a WAL track data updates and their ordering.
- [**Read Log**](docs/concepts.md#read-log): logs all read operations, with their originator.
- [**Authentication**](docss/auth.md): datamon keeps track of who contributed what, when and in which order (WAL) and who accessed what (Read Log).

## Installation

Please follow the [installation instructions](docs/install.md).

## CLI Guide

Datamon comes as a CLI tool: see [usage](docs/usage/datamon.md).

## Use cases

* ARGO ML pipeline
* [Datamover container guide](docs/datamover.md)
* [Datamon as sidecar](docs/sidecar.md)
* [Kubernetes integration](docs/kubernetes.md)

## Feature requests and bugs

Please file [GitHub issues](https://github.com/oneconcern/datamon/issues) for feature requests or bug reports.

## Contributing

Please read our [contributing guidelines](CONTRIBUTING.md)

## License

Datamon is developed by [OneConcern Inc.](https://wwww.oneconcern.com) under the [MIT license](LICENSE).
