# Concepts

## Repo 

A datamon repository is analogous to a git repo. A repo is a dataset that has a unified lifecycle.
A particular version of the files in a repo is called a [`bundle`](#bundle).

See [usage](usage/datamon_repo.md)

## Bundle 

A bundle is a point in time read-only view of a repo, composed of individual files. This is analogous to a commit in git.

See [usage](usage/datamon_bundle.md)

Bundle metadata can be _downloaded_ (e.g. to see contributors), bundle blob content can be _mounted_ on a host mount path.

A simple read-only mount provide a file system view of the bundle content.

A _new mount_ will mount a mutable view of the bundle.

Eventually, any set of files from the local filesystem (for instance, a _modified_ version of a mutable mount),
may be _uploaded_ to an archived state.

## Label 

A name given to a bundle, analogous to tags in git. Examples: Latest, production.

> NOTE: at this moment, tags a simple strings and cannot be annotated or signed like git tags.

See [usage](usage/datamon_label.md)

## Context

A [context](context.md) provides a way to define multiple instances of datamon.

See [usage](usage/datamon_context.md)

## Write Ahead Log

The [WAL](proposals/wal.md) tracks data updates and their ordering.

## Read Log

The **Read Log** logs all read operations, with their originator.
