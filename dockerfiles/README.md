# datamon docker files

## datamon

**Image**: `gcr.io/onec-co/datamon`

[Dockerfile](./datamon.Dockerfile)

Statically linked datamon binary (built on Alpine).

## Data migration tool

**Image**: `gcr.io/onec-co/datamon-migrate`

[Dockerfile](./migrate.Dockerfile)

Statically linked datamon migrate utility (built on Alpine).

Ships with the regular datamon binary.

## Sidecars

Sidecar images are based on latest debian. A non-root user "developer" is defined to run
non-privileged sidecar processes.

**Image**: `gcr.io/onec-co/datamon-fuse-sidecar`

[Dockerfile](./sidecar.Dockerfile)

**Image**: `gcr.io/onec-co/datamon-pg-sidecar`

[Dockerfile](./sidecar-pg.Dockerfile)

**Image**: `gcr.io/onec-co/datamon-wrapper`

Only contains the coordination wrapper script to use with datamon sidecars (Bourne shell)

[Dockerfile](./wrapper.Dockerfile)

## Base images

Base images updated at every PR run (when PRs are more frequent, consider moving this back to a weekly CI cronjob).

Notice that these updates are not picked automatically to build released images.
This is on purpose: we don't want to propagate dependency updates untested (e.g. new postgres releases).

Sidecar building base:
[Dockerfile](./sidecar-base.Dockerfile)

pg sidecar building base:
[Dockerfile](./pgsidecar-base.Dockerfile)


## CI builder image

Our CI is largely using a pre-baked docker executor based on a base debian golang image `golang:1.16`.
This is updated at every PR run. CI always runs on the latest image (when PRs are more frequent, consider moving this back to a weekly CI cronjob).

Notice: circleci convenience images have not proved to be much more efficient or downloaded significantly faster. These are mostly a pain: moving forward, use standard golang images.

[Dockerfile](./builder.Dockerfile)

## Other docker builds

Alpine base: an intermediary build used when compiling statically linked binaries.

[Dockerfile](./alpine-base.Dockerfile)

Binaries: an intermediary build used to compile all helper binaries used by sidecars.

[Dockerfile](./binaries-base.Dockerfile)
