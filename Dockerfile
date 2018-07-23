FROM golang:alpine as base

RUN mkdir -p /stage/data /stage/etc/ssl/certs &&\
  apk add --no-cache musl-dev gcc ca-certificates mailcap upx tzdata zip &&\
  update-ca-certificates &&\
  cp /etc/ssl/certs/ca-certificates.crt /stage/etc/ssl/certs/ca-certificates.crt &&\
  cp /etc/mime.types /stage/etc/mime.types

WORKDIR /usr/share/zoneinfo
RUN zip -r -0 /stage/zoneinfo.zip .

ADD . /go/src/github.com/oneconcern/trumpet
WORKDIR /go/src/github.com/oneconcern/trumpet

RUN go build -o /stage/usr/bin/tpt --ldflags '-s -w -linkmode external -extldflags "-static"' ./cmd/tpt
RUN upx /stage/usr/bin/tpt

# Build the dist image
FROM scratch
COPY --from=base /stage /
ENV ZONEINFO /zoneinfo.zip
ENTRYPOINT [ "tpt" ]
CMD ["--help"]
