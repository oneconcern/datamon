# Pack all datamon binaries in one single image (used during CI testing: this image is not released)
#
FROM reg.onec.co/datamon-alpine-base:latest as base

ARG version
ARG commit
ARG dirty

ENV VERSION ${version}
ENV GIT_COMMIT ${commit}
ENV GIT_DIRTY ${dirty}
ENV IMPORT_PATH github.com/oneconcern/datamon/cmd/datamon/cmd

WORKDIR /build
RUN make cross-compile-binaries TARGET=/stage/usr/bin OS=linux  && \
    bash -c 'cd /stage/usr/bin ; for bin in $(ls -1) ; do mv ${bin} ${bin%_linux_amd64} ;done' && \
    ln /stage/usr/bin/backup2blobs /stage/usr/bin/migrate

FROM scratch
COPY --from=base /stage /stage
