FROM golang:alpine as base

RUN mkdir -p /stage/usr/bin &&\
  apk add --no-cache musl-dev gcc ca-certificates mailcap upx

ADD . /go/src/github.com/oneconcern/trumpet
WORKDIR /go/src/github.com/oneconcern/trumpet

RUN cp ./flexvoldrivers/deploy.sh /stage/usr/bin/

## build goofys driver
RUN go build -o /stage/flexgoofys --ldflags '-s -w -linkmode external -extldflags "-static"' ./flexvoldrivers/flexgoofys
RUN upx /stage/flexgoofys

# Build the dist image
FROM bash
COPY --from=base /stage /
CMD "/usr/bin/deploy.sh"
