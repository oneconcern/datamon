# Build environment
FROM golang:1.10-alpine as base

RUN mkdir -p /stage/data /stage/usr/bin /stage/etc/ssl/certs &&\
  apk add --no-cache musl-dev gcc ca-certificates mailcap upx git

RUN go get -d github.com/kahing/goofys
WORKDIR /go/src/github.com/kahing/goofys

RUN go build -o /stage/usr/bin/goofys --ldflags '-s -w -linkmode external -extldflags "-static"' .
RUN upx /stage/usr/bin/goofys



# Build the dist image
FROM python:alpine
COPY --from=base /stage /
RUN apk add --no-cache ca-certificates mailcap tzdata fuse && pip install awscli
CMD ["goofys", "--help"]


