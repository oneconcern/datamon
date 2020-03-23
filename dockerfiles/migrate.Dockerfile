# migrate tool
FROM reg.onec.co/datamon-alpine-base:latest as base

ARG version
ARG commit
ARG dirty

ENV VERSION ${version}
ENV GIT_COMMIT ${commit}
ENV GIT_DIRTY ${dirty}
ENV IMPORT_PATH github.com/oneconcern/datamon/cmd/datamon/cmd

# TODO(fred): use link flags to version migrate tool
WORKDIR /build
RUN LDFLAGS='-s -w -linkmode external -extldflags "-static"' && \
    LDFLAGS="$LDFLAGS -X '${IMPORT_PATH}.Version=${VERSION}'" && \
    LDFLAGS="$LDFLAGS -X '${IMPORT_PATH}.BuildDate=$(date -u -R)'" && \
    LDFLAGS="$LDFLAGS -X '${IMPORT_PATH}.GitCommit=${GIT_COMMIT}'" && \
    LDFLAGS="$LDFLAGS -X '${IMPORT_PATH}.GitState=${GIT_DIRTY}'" && \
    go build -o /stage/usr/bin/migrate --ldflags '-s -w -linkmode external -extldflags "-static"' ./cmd/backup2blobs/ &&\
    upx /stage/usr/bin/migrate &&\
    md5sum /stage/usr/bin/migrate

#Build the dist image
FROM scratch
COPY --from=base /stage /
ENV ZONEINFO /zoneinfo.zip
ENTRYPOINT [ "/usr/bin/migrate" ]
CMD [ "--help" ]
