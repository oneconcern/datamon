FROM circleci/golang

RUN sudo apt-get update -y
RUN sudo apt-get install lsb-release
RUN echo -n "cloud-sdk-$(lsb_release -c -s)" > /tmp/CLOUD_SDK_REPO
RUN echo "deb http://packages.cloud.google.com/apt $(cat /tmp/CLOUD_SDK_REPO) main" \
  | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list
RUN curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add
RUN sudo apt-get update -y
RUN sudo apt-get install -y git build-essential
RUN sudo apt-get install -y google-cloud-sdk
RUN sudo apt-get install -y shellcheck



ENTRYPOINT [ "/bin/sh"]
CMD ["sh"]
