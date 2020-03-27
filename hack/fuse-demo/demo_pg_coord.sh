#! /bin/bash
#
# A sample bootstrap script to run the k8s demo.
#
# It winds up some postgres databases from a datamon bundle, save them, then reuse them.

set -e -o pipefail
ROOT="$(git rev-parse --show-toplevel)"
cd "${ROOT}"||exit 1
# shellcheck disable=SC1091
. ./hack/fuse-demo/consts_demo_pg.sh
# shellcheck disable=SC1091
. ./hack/fuse-demo/funcs_demo_pg.sh
LOG_FILE="/tmp/datamon-coord-pg-demo.log"

# Whether we rebuild the things we need before starting.
# By default, we assume the building job as already been carried out.
build_sidecar=false
build_demo_app=false
build_datamon_base=false

while getopts bsc: opt; do
    case $opt in
        b)
            build_datamon_base=true
            ;;
        s)
            build_sidecar=true
            build_demo_app=true
            ;;
        *)
            echo "Usage: demo_pg_coord.sh [-b][-s]"
            echo
            echo "-b               rebuild the container with datamon binaries"
            echo "-s               rebuild containers for sidecar and demo mock app"
            exit 1
            ;;
    esac
done
(( OPTIND > 1 )) && shift $(( OPTIND - 1 ))

if [[ "${build_datamon_base}" == "true" ]]; then
    make build-datamon-binaries
fi
if [[ "${build_sidecar}" == "true" ]] ; then
    make build-and-push-pg-sidecar
fi
if [[ "${build_demo_app}" == "true" ]] ; then
    make pg-demo-coord-build-app
fi

# determine current branch tag (used to daisy chain builds when run by CI)
SIDECAR_TAG=$(go run ./hack/release_tag.go)
info_print "running demo built with image TAG: $SIDECAR_TAG"
info_print "creating coordinator pod"
./hack/fuse-demo/create_coord_pg_pod.sh -t "${SIDECAR_TAG}"

pod_name=""
info_print "waiting on pod start"
typeset -i COUNT
COUNT=0
while [[ -z "${pod_name}" ]]; do
    sleep "${POLL_INTERVAL}"
    if ! k=$(kubectl -n "${NS}" get pods -l app=datamon-coord-pg-demo,instance="${SIDECAR_TAG}" --output custom-columns=NAME:.metadata.name,STATUS:.status.phase) ; then
      # sometimes, we lose connectivity from the circleCI container: handle failure and retry a couple times
      error_print "cannot fetch pod logs. Retrying..."
      COUNT=$((COUNT+1))
      if [[ "${COUNT}" -gt 10 ]] ; then
        error_print "cannot fetch pod logs. Giving up after ${COUNT} attempts."
        exit 1
      fi
      continue
    fi
    pod_name=$(echo "${k}" | grep Running | cut -d' ' -f1) || true
    check=$(echo "${k}"|grep -iE '(BackOff)|(Error)') || true
    if [[ -n "${check}" ]] ; then
      error_print "pod won't start. Exiting..."
      error_print "${k}"
      exit 1
    fi
done

info_print "pod started, following logs from $pod_name"
rm -f "${LOG_FILE}" && touch "${LOG_FILE}"
(sleep 600 && echo "timed out on waiting" && kill -15 $$ || exit 1) & # stops waiting after 10m
(kubectl -n "${NS}" logs -f --all-containers=true "${pod_name}"|tee -a "${LOG_FILE}" || true) &
# now wait for the demo to complete
(tail -f "${LOG_FILE}" || true)|while read -r line ; do
  if [[ "${line}" =~ "application exited with non-zero-status" ]] ; then
    error_print "error detecting in a container"
    exit 1
  fi
  if [[ "${line}" =~ "wrap_application sleeping indefinitely" ]] ; then
    break
  fi
done
info_print "success: wrapper ended successfully"
exit 0
