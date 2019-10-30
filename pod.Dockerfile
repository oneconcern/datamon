FROM golang:alpine as base
RUN apk add --no-cache --quiet \
      musl-dev gcc ca-certificates mailcap \
      upx tzdata zip git bash zsh tcsh && \
    update-ca-certificates

FROM base as builder
ARG version
ARG commit
ARG dirty

ENV VERSION ${version}
ENV GIT_COMMIT ${commit}
ENV GIT_DIRTY ${dirty}

ADD . /datamon
WORKDIR /datamon

RUN LDFLAGS='-s -w -linkmode external -extldflags "-static"' && \
  LDFLAGS="$LDFLAGS -X 'github.com/oneconcern/datamon/cmd/datamon/cmd.Version=${VERSION}'" && \
  LDFLAGS="$LDFLAGS -X 'github.com/oneconcern/datamon/cmd/datamon/cmd.BuildDate=$(date -u -R)'" && \
  LDFLAGS="$LDFLAGS -X 'github.com/oneconcern/datamon/cmd/datamon/cmd.GitCommit=${GIT_COMMIT}'" && \
  LDFLAGS="$LDFLAGS -X 'github.com/oneconcern/datamon/cmd/datamon/cmd.GitState=${GIT_DIRTY}'" && \
  go version && \
  mkdir -p /stage/usr/bin && \
  go build -o /stage/usr/bin/datamon --ldflags "$LDFLAGS" ./cmd/datamon

RUN upx /stage/usr/bin/datamon

# TODO: we could propose other bases than alpine, esp. if people
# want to interact with their favorite language, e.g. python, etc.
# The alpine-built datamon binary should be able to run on any amd64 arch.
FROM base
COPY --from=builder /stage/usr/bin/datamon /usr/bin/datamon
WORKDIR /home/project
ENTRYPOINT [ "bash" ]
