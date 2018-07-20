# List of open todos

This repo is currently a Proof Of Concept of how the trumpet tool would work.

## Garbage collection

Garbage collection should work out which commits are no longer referenced.
From there it needs needs to work out which blobs are no longer in use and prune those blobs.
When the blobs are pruned the bundles that represent the commits are removed.

## Branch checkout

A branch checkout doesn't currenlty do anything except for dump the snapshot that would be restored

## Workspace concept

A repository should get a workspace concept. This is the actual manifestation of the snapshots on disk.
A workspace will be used to distinguish between untracked files, how we can do update detection etc.

With a workspace, anything that will get added to the repo has to be present in the workspace

## better logging

Currently there is almost no debug logging, so when something goes wrong it will be hard to know what happened

## tests

There are no tests for any of this.

## client/server

As initial POC this is implemented as a CLI application that uses the local fs as a backing store.

## Index of commit to repository mapping

The repository store needs to track all the known commits with their repository names
