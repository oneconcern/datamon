FROM circleci/golang
ENV GO111MODULE on
RUN sudo apt-get update -y --quiet
RUN sudo apt-get install -y --quiet lsb-release upx git build-essential shellcheck zsh
RUN export CLOUD_SDK_REPO="cloud-sdk-$(lsb_release -c -s)" && \
    echo "deb http://packages.cloud.google.com/apt $CLOUD_SDK_REPO main" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && \
    curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add && \
    sudo apt-get update -y --quiet && \
    sudo apt-get install -y --quiet google-cloud-sdk kubectl

RUN go get -u gotest.tools/gotestsum@latest && \
    go get -u github.com/matryer/moq@latest && \
    go get -u github.com/golangci/golangci-lint/cmd/golangci-lint@latest && \
    go clean -modcache
