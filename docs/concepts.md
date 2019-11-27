# Concepts

## Repo 

A datamon repository is analogous to a git repo. A repo is a dataset that has a unified lifecycle.
A particular version of the files in a repo is called a [`bundle`](#bundle).

```
Usage:
  datamon repo [command]

Available Commands:
  create      Create a named repo
  get         Get repo info by name
  list        List repos

Flags:
  -h, --help   help for repo
```

## Bundle 

A bundle is a point in time read-only view of a repo, composed of individual files. This is analogous to a commit in git.

```
Usage:
  datamon bundle [command]

Available Commands:
  diff        Diff a downloaded bundle with a remote bundle.
  download    Download a bundle
  get         Get bundle info by id
  list        List bundles
  mount       Mount a bundle
  update      Update a downloaded bundle with a remote bundle.
  upload      Upload a bundle

Flags:
  -h, --help   help for bundle
```

Bundle metadata can be _downloaded_ (e.g. contributors), bundle blob content can be _mounted_ on a host mount path.

A simple read-only mount provide a file system view of the bundle content.

A _new mount_ will mount a mutable view of the bundle.

Eventually, any set of files from the local filesystem (for instance, a _modified_ version of a mutable mount),
may be _uploaded_ to an archived state.

## Label 

A name given to a bundle, analogous to tags in git. Examples: Latest, production.

> NOTE: at this moment, tags a simple strings and cannot be annotated or signed like git tags.

```
Usage:
  datamon label [command]

Available Commands:
  get         Get bundle info by label
  list        List labels
  set         Set labels

Flags:
  -h, --help   help for label
```

## Context

A [context](context.md) provides a way to define multiple instances of datamon.

```
Usage:
  datamon context [command]

Available Commands:
  create      Create a context
```

## Write Ahead Log

The [WAL](wal.md) tracks data updates and their ordering.

## Read Log

The **Read Log** logs all read operations, with their originator.
