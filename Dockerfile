FROM golang:alpine as base

RUN mkdir -p /stage/data /stage/usr/bin /stage/etc/ssl/certs &&\
  apk add --no-cache musl-dev gcc ca-certificates mailcap upx

ADD . /go/src/github.com/oneconcern/trumpet
WORKDIR /go/src/github.com/oneconcern/trumpet

RUN go build -o /stage/usr/bin/tpt --ldflags '-s -w -linkmode external -extldflags "-static"' ./cmd/tpt
RUN upx /stage/usr/bin/tpt

# Build the dist image
FROM alpine
COPY --from=base /stage /
RUN apk add --no-cache ca-certificates mailcap tzdata
CMD ["tpt", "--help"]
