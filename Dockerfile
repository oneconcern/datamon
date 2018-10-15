FROM golang:alpine as base

ARG github_user
ARG github_token

ENV GITHUB_USER ${github_user}
ENV GITHUB_TOKEN ${github_token}

ADD hack/create-netrc.sh /usr/bin/create-netrc

RUN mkdir -p /stage/data /stage/etc/ssl/certs &&\
  create-netrc &&\
  apk add --no-cache musl-dev gcc ca-certificates mailcap upx tzdata zip git &&\
  update-ca-certificates &&\
  cp /etc/ssl/certs/ca-certificates.crt /stage/etc/ssl/certs/ca-certificates.crt &&\
  cp /etc/mime.types /stage/etc/mime.types

WORKDIR /usr/share/zoneinfo
RUN zip -r -0 /stage/zoneinfo.zip .

ADD . /datamon
WORKDIR /datamon

RUN go build -o /stage/usr/bin/datamon --ldflags '-s -w -linkmode external -extldflags "-static"' ./cmd/datamon
RUN upx /stage/usr/bin/datamon

# Build the dist image
FROM scratch
COPY --from=base /stage /
ENV ZONEINFO /zoneinfo.zip
ENTRYPOINT [ "datamon" ]
CMD ["--help"]

