[![CircleCI](https://circleci.com/gh/oneconcern/datamon/tree/master.svg?style=svg&circle-token=e827ee1509892d8ba85db2a819b692ca451a7a97)](https://circleci.com/gh/oneconcern/datamon/tree/master)
[![GitHub release](https://img.shields.io/github/v/release/oneconcern/datamon)](https://github.com/oneconcern/datamon/releases/latest)
[![license](https://img.shields.io/badge/license-MIT-green)](https://raw.githubusercontent.com/oneconcern/datamon/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/oneconcern/datamon?status.svg)](http://godoc.org/github.com/oneconcern/datamon)

# Datamon

Datamon is a data science tool sponsored by [OneConcern](https://www.oneconcern.com) that helps manage data at scale.

## Primer

### Goals

The primary goal of datamon is to manage versioned data at rest, providing CLI tools for creation, access and tracking
in an environment where data repositories and their lifecycles are linked.

Datamon links the various sources of data, how they are processed and tracks the
output/new data that is generated from the existing data.

[More on design and architecture](docs/design.md).

### Features

* Manage data sets as versioned repositories stored on a cloud storage backend
* Manage metadata for these data sets (versions, labels, file sets...)
* Multi-tenancy using contexts
* Lineage tracking backed by cloud authentication
* Store data sets as fixed size deduplicated blobs, using blake hashing
* Versions ("bundles") may be uploaded then downloaded on local storage
* Versions may be accessed directly on a mounted file system (fuse)
* CLI management tool
* Metrics collection

### Added value

* Leverages low-cost frozen storage (e.g. S3, GCS)
* Optimized billed operations for storage: no fancy billable backend store options are used (like concurrency control, etc)
* Optimized for speed: parallel I/Os together with deduplication vastly outperform usual tools like `gsutil`
* A well-defined and tested immutable metadata model ensures that no data is ever lost or unrecoverable. Datamon is an effective substitute to many bespoke gsutil scripting utilities.
* Versioning & tagging occur on whole data sets and not individual files. This makes it easy to restore consistent inputs to some reproducible computation
* Less storage bucket administration: datamon uses only a few buckets, defined according to IAM policies (i.e. a datamon context)
* Repositories make up a convenient abstration for datasets, and share the same underlying cloud storage bucket configuration (abstracted as a "context")

#### Extra tools

* Scripted interface to use as a sidecar container (e.g. for ARGO workflows)

#### Experimental

* Mutable fuse mount, to commit versioned data sets directly from a mounted file system

#### Coming soon...

* [X] Diamond workflow: several collaborating nodes produce a versioned dataset in parallel
* [ ] Python bindings
* [ ] Write Ahead / Read Ahead logs

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
- [**Write Ahead Log**](docs/concepts.md#write-ahead-log): a WAL tracks data updates and their ordering.
- [**Read Log**](docs/concepts.md#read-log): logs all read operations, with their originator.
- [**Authentication**](docs/auth.md): datamon keeps track of who contributed what, when and in which order (WAL) and who accessed what (Read Log).

## Installation

Please follow the [installation instructions](docs/install.md).

## Migrating from v1 to v2

v2 comes with breaking changes. The migration process replaces older repos by new ones.

See the [migration guide](k8s/migatev2/README.md).

## CLI guide

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
