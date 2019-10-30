#!/bin/bash
# TODO: this is different on macs
CONFIG_DIR=${HOME}/.config/gcloud
kubectl create secret generic my-creds --from-file=${CONFIG_DIR}/application_default_credentials.json
