FROM golang:alpine as base
RUN apk add --no-cache --quiet \
      musl-dev gcc ca-certificates mailcap \
      upx tzdata zip openssh git bash zsh tcsh && \
    update-ca-certificates

FROM base as builder
ARG version
ARG commit
ARG dirty

ENV VERSION ${version}
ENV GIT_COMMIT ${commit}
ENV GIT_DIRTY ${dirty}

ENV GO111MODULE on
ENV GOBIN ${GOPATH}/bin
RUN mkdir -p ${GOBIN} && mkdir -p /stage/usr/bin

WORKDIR ${GOPATH}/src/github.com/oneconcern
RUN git clone https://github.com/oneconcern/datamon datamon1 && cd datamon1 && git checkout v1.0.0
RUN cd ${GOPATH}/src/github.com/oneconcern/datamon1/cmd/datamon && go build && upx datamon && mv datamon /stage/usr/bin/datamon1

WORKDIR ${GOPATH}/src/datamon2
RUN go get -tags bundle_preserve github.com/oneconcern/datamon/cmd/datamon@master
RUN upx $GOBIN/datamon && mv $GOBIN/datamon /stage/usr/bin/datamon2

FROM base
COPY --from=builder /stage/usr/bin/datamon1 /usr/bin/datamon1
COPY --from=builder /stage/usr/bin/datamon2 /usr/bin/datamon2
WORKDIR /home/project
RUN mkdir -p /data
ENV HOME /home/project
ENV GOOGLE_APPLICATION_CREDENTIALS /home/project/.config/gcloud/application_default_credentials.json
ENTRYPOINT [ "bash" ]
