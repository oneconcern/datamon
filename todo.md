# List of open todos

This repo is currently a Proof Of Concept of how the datamon tool would work.

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

## Ignore file support

A data repository should support a .datamonignore file

This seems like it would work: https://github.com/codeskyblue/dockerignore

## better logging

Currently there is almost no debug logging, so when something goes wrong it will be hard to know what happened

## tests

There are no tests for any of this.

## client/server

As initial POC this is implemented as a CLI application that uses the local fs as a backing store.

## Index of commit to repository mapping

The repository store needs to track all the known commits with their repository names

## Config directories

Use application specific config directories.

### OSX

This means asking for the Application Support directory: 

* [Code Sample for Go](https://coderwall.com/p/l9jr5a/accessing-cocoa-objective-c-from-go-with-cgo)
* [Apple Developer Guide](https://developer.apple.com/library/archive/documentation/General/Conceptual/MOSXAppProgrammingGuide/AppRuntime/AppRuntime.html#//apple_ref/doc/uid/TP40010543-CH2-SW9)

### Linux

This means using the contents of `$XDG_CONFIG_DIRS`: 

* [Arch Linux Wiki](https://wiki.archlinux.org/index.php/XDG_Base_Directory_support)
* [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html)

### Windows

%APPDATA%

C:\Users\username\AppData\Roaming

* [MSDN folderid](https://docs.microsoft.com/en-us/windows/desktop/shell/knownfolderid)
* [Pointlogic appdata folder](https://support.pointlogic.com/faq/troubleshooting/accessing-the-appdata-folder)
