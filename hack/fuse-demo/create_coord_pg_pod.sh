#! /bin/bash
#
# A utility script to create a k8 pod with the pg demo app
#
# This script takes some parameters as env vars and expands this to create a k8 template
# (e.g. a very simplified templating mechanism).
#
# Requires:
#  - git
#  - go
#  - ncurses
#  - kubectl
#  - env GOOGLE_APPLICATION_CREDENTIALS set

set -e -o pipefail
ROOT="$(git rev-parse --show-toplevel)"
cd "${ROOT}/hack/fuse-demo" || exit 1
# shellcheck disable=SC1091
. ./consts_demo_pg.sh
# shellcheck disable=SC1091
. ./funcs_demo_pg.sh

EXPAND_CMD=(go run "${ROOT}/hack/envexpand.go")
POD_TEMPLATE="${ROOT}/hack/k8s/example-coord-pg.template.yaml"

if [[ -z "${GOOGLE_APPLICATION_CREDENTIALS}" ]]; then
	error_print "GOOGLE_APPLICATION_CREDENTIALS env variable not set" 1>&2
  exit 1
fi

while getopts ot: opt; do
    case $opt in
        o)
            # local deploy
            PULL_POLICY=IfNotPresent
            ;;
        t)
            SIDECAR_TAG="$OPTARG"
            ;;
        *)
            echo "Usage: create_coord_pg_pod.sh [-o][-t {tag}]"
            echo
            echo "-o               set image pull policy to IfNotPresent (e.g. for local cluster not authenticated to gcr.io registry)"
            echo "-t               set image tag to deploy"
            exit 1
            ;;
    esac
done
(( OPTIND > 1 )) && shift $(( OPTIND - 1 ))

info_print "building k8s demo with tag: $SIDECAR_TAG"

if kubectl -n "${NS}" get deployment "${DEPLOYMENT_NAME}-${SIDECAR_TAG}" &> /dev/null; then
	kubectl -n "${NS}" delete deployment "${DEPLOYMENT_NAME}-${SIDECAR_TAG}"
fi
for i in "1" "2" "3"; do
  if kubectl -n "${NS}" get configmap "${BASE_CONFIG_NAME}-${i}-${SIDECAR_TAG}" &> /dev/null; then
	  kubectl -n "${NS}" delete configmap "${BASE_CONFIG_NAME}-${i}-${SIDECAR_TAG}"
  fi
done
# TODO(fred): I don't think this is needed
if kubectl -n "${NS}" get secret google-application-credentials &> /dev/null; then
	kubectl -n "${NS}" delete secret google-application-credentials
fi

kubectl -n "${NS}" create secret generic \
	google-application-credentials \
	--from-file=google-application-credentials.json="${GOOGLE_APPLICATION_CREDENTIALS}"

# resolve template
RES_DEF="/tmp/example-coord-pg.yaml"

export PULL_POLICY OUTPUT_LABEL EXAMPLE_DATAMON_REPO SIDECAR_TAG NS INPUT_LABEL_2 INPUT_LABEL_3
export DEPLOYMENT_NAME="${BASE_DEPLOYMENT_NAME}-${SIDECAR_TAG}"
export CONFIG_NAME_1="${BASE_CONFIG_NAME}-1-${SIDECAR_TAG}"
export CONFIG_NAME_2="${BASE_CONFIG_NAME}-2-${SIDECAR_TAG}"
export CONFIG_NAME_3="${BASE_CONFIG_NAME}-3-${SIDECAR_TAG}"
"${EXPAND_CMD[@]}" "${POD_TEMPLATE}" > "${RES_DEF}"

dbg_print "pod template $(cat "${RES_DEF}")"

# deploy demo pod
if ! kubectl -n "${NS}" create -f "${RES_DEF}" ; then
  error_print "failed to create demo pod"
  exit 1
fi
info_print "pod created"
