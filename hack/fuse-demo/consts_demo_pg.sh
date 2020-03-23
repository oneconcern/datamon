#! /bin/bash
#
# Constants defined for the demo
# default values
#
export POLL_INTERVAL=1 # seconds  -- during wait loops

# kubernetes template settings
export NS=datamon-ci  # namespace used for this demo
export INPUT_LABEL_2=pg-coord-initial
export INPUT_LABEL_3=pg-coord-frozen
export OUTPUT_LABEL=pg-coord-example # the default label when saving a modified database as a datamon bundle
export SIDECAR_TAG=latest
export PULL_POLICY=Always
export BASE_DEPLOYMENT_NAME=datamon-pg-demo
export BASE_CONFIG_NAME=datamon-pg-demo-config
export EXAMPLE_DATAMON_REPO=datamon-pg-test-repo
