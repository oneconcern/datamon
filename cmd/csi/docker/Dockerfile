FROM golang:1.11.1-stretch as builder

WORKDIR /go/src/github.com/oneconcern/csi-gce

ADD . .

FROM ubuntu:16.04
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates wget autofs \
        && echo "deb http://packages.cloud.google.com/apt cloud-sdk-xenial main" | tee /etc/apt/sources.list.d/google-cloud.sdk.list \
        && echo "deb http://packages.cloud.google.com/apt gcsfuse-xenial main" | tee /etc/apt/sources.list.d/gcsfuse.list \
        && wget -qO- https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add - \
        && apt-get update && apt-get install -y --no-install-recommends google-cloud-sdk gcsfuse make \
        && mkdir -p /etc/autofs && touch /etc/autofs/auto.gcsfuse && rm -rf /var/lib/apt/lists


#COPY --from=builder /go/src/github.com/chandresh-pancholi/csi-gce/application_default_credentials.json .config/gcloud/application_default_credentials.json

COPY --from=builder /go/src/github.com/oneconcern/csi-gce/csi /csi

COPY --from=builder /go/src/github.com/oneconcern/csi-gce/cred.json /cred.json

ENV GOOGLE_APPLICATION_CREDENTIALS=/cred.json

RUN chmod +x /csi

ENTRYPOINT ["/csi"]
