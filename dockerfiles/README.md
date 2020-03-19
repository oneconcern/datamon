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

Base images updated every week. Notice that these updates are not picked automatically to build released images.
This is on purpose: we don't want to propagate dependency updates untested (e.g. new postgres releases).

Sidecar building base:
[Dockerfile](./sidecar-base.Dockerfile)

pg sidecar building base:
[Dockerfile](./pgsidecar-base.Dockerfile)


## CI builder image

Our CI is largely using a pre-baked docker executor based on circle convenience image `cimg/go`.
This is updated every week and CI always runs on the latest image.

Notice that at the moment `cimgo/go` doesn't have a `latest` tag, so we are left with specifying the go version
explicitly (now `1.14`).

[Dockerfile](./builder.Dockerfile)

## Other docker builds

Alpine base: an intermediary build used when compiling statically linked binaries.

[Dockerfile](./alpine-base.Dockerfile)

Binaries: an intermediary build used to compile all helper binaries used by sidecars.

[Dockerfile](./binaries-base.Dockerfile)
