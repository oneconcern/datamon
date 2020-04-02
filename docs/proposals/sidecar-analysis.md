# Sidecar analysis

## Primer

Datamon sidecars come in two flavors: fuse mount and postgres.

## Lessons learned

Any of the implemented sidecars:
* Only supports one linear workflow: load, update, save
* Has most of its code and complexity for parameters handling
* Is a zsh daemon shell script.
  There is not real justification for using this uncommon shell.
  This seemed to boil down to the use of associative arrays, which also work in bash.
* Relies on a central bourne shell script (wrapper) to operate the load/save workflow with datamon

CI is also a concern:
* These are very long to build (builds many times the same thing)
* Sidecar "demo" jobs do not play well with CI (long builds, long runs, sensitive to parallel runs, possible infinite loops, does not relinquish k8s resources)
* No linting job supported on zsh

Current pg sidecar:
* is essentially bereft of practical usage documentation
* is _very_ difficult to parameterize, with no sensible default values
* was designed to run _several_ databases in a single container
* was made essentially for demo
* the raw demo essentially does nothing and needs a lot of extra parameters to demonstrate anything useful
* has many untested paths, which don't work
* ignores most postgres utilities and manages processes directly
* relies on datamon fuse mount (performance, and questionable support for all syscalls from the database)
* makes up a tar archive of the raw dabase files, doubling the required storage to prepare an upload
* the k8 demo parameterization job is basically a bespoke helm (e.g. replacing env in template)
* contains a lot of stuff about running non-root, but this eventually doesn't work and is not used
* is not aware of datamon changes regarding auth (email, user in config)
* has no solution for database version upgrades

Current fuse sicar:
* relies on a specific binary sidecar_param, to unfold a yaml document as env variables

## Short term corrective actions

We cannot afford a full rewrite in the short term.
We want to avoid altering the fuse sidecar as much as possible since that one is allegedly working.

I want to alleviate much of our CI issues that come from the sidecar demos (long runs, CI credits consumption, unstable behavior)
AND to actually demonstrate a workable postgres sidecar. This lead to a couple thoughts described [here](sidecar-design.md).

Here are the many minor fixes brought by PR#428.

* CI:
  * Overhauled Dockerfiles and image tagging: CI can build faster
  * All images are debian based, with same version ; postgres is installed locally from a debian package, not a docker image
  * Removed redundant builds during CI jobs
  * Added trailing CI tasks to ensure k8s resources are relinquished
  * Changed the layout of the CI workflow to avoid starting the k8 jobs too early
  * Every CI run creates kubernetes resources which are unique to this run
  * ~~Convenience builder images are updated on a weekly basis~~ (removed)
  * added IP whitelisting to deploy k8s demos on 1C cluster

* pg k8 demo
  * Replaced env-based parameterization by k8s configmap objects and YAML parsing inside the sidecar
  * Removed many demo-specific options

* pg sidecar
  * Ported all pg-related scripts to bash - these scripts are now covered by linter checks
  * Introduced sensible defaults for most parameters (default to one single db, mount point, port number, locations, etc)
  * Covered the use case of thrown away, read-only databases (you just don't specify any destination repo)
  * Used standard postgres start/stop utilities. Removed the bespoke PID handling.
  * Adapted to run as non-root, as initially intended, but unfinished
  * Made any pg sidecar run only one database server (a server can publish several databases): parameterization is simpler
  * Removed most of the pg parameterization logic and replaced this by a (publicly available) basic yaml parser for shell
  * Adapted coordination logic to support several pg sidecars from the same application wrapper
  * Removed fuse mount usage for postgres and replaced by download/upload
  * Removed usage of tar and copy of database files: download/upload operations are carried out in place
  * Introduced different test scenarios in the mock application

* fuse sidecar
  * Adapted k8 template to use dedicated namespace and create k8 resources that are unique to a CI run
  * Phased out demo-specific sidecar image: demo runs on regular builds

## Longer term corrective actionss

Despite its weight, PR#428 has left behind quite a number of unsolved issues.

* [ ] 12 factor-app parameterization, which is difficult to achieve in shell and easy in go (some would prefer python, though)
* [ ] Sidecar_param binary is essentially overlapping with golang sp13/viper: a golang-based sidecar wouldn't need that, just viper
* [ ] Adapt wrapper logic to support many sidecars, including a mix of fuse & postgres ones
* [ ] Adapt wrapper logic to support other workflows, such as "only read, don't update"
* [ ] Take actual provisions to handle postgres version migrations
* [ ] Handle errors gracefully & allow for out of band signaling: the "application wrapper" should stop when sidecars fail
