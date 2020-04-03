#! /bin/zsh

setopt ERR_EXIT

dbg_print() {
    local COL_GREEN
    local COL_RESET
    COL_YELLOW=$(tput -Txterm setaf 3)
    COL_RESET=$(tput -Txterm sgr0)
    echo ${COL_YELLOW}
    print -- $1
    echo ${COL_RESET}
}

#

SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"
NS=datamon-ci

proj_root_dir="$(dirname "$(dirname "$SCRIPT_DIR")")"

pull_policy=Always
shell_only=false

while getopts So opt; do
    case $opt in
        (o)
            # local deploy
            pull_policy=IfNotPresent
            ;;
        (S)
            # local deploy
            shell_only=true
            ;;
        (\?)
            print Bad option, aborting.
            exit 1
            ;;
    esac
done
(( OPTIND > 1 )) && shift $(( OPTIND - 1 ))

dbg_print 'create_coord_pod.sh getopts end'

if ! kubectl -n "${NS}" get secret google-application-credentials &> /dev/null; then
  kubectl -n "${NS}" create secret generic \
	  google-application-credentials \
	  --from-file=google-application-credentials.json="${GOOGLE_APPLICATION_CREDENTIALS}"
fi

##

TEMPLATE_NAME='example-coord'
if $shell_only; then
    TEMPLATE_NAME='example-coord_shell-only'
fi
# determine current branch tag (used to daisy chain builds when run by CI)
SIDECAR_TAG=$(go run ./hack/release_tag.go)
dbg_print "running demo built with image TAG: $SIDECAR_TAG"

DEPLOYMENT_NAME="datamon-fuse-demo-${SIDECAR_TAG}"
RES_DEF="/tmp/${TEMPLATE_NAME}.yaml"
dbg_print "### templating k8s api server yaml for kubectl -n $NS cmd to ${RES_DEF}"

PULL_POLICY=$pull_policy \
DEPLOYMENT_NAME="${DEPLOYMENT_NAME}" \
SIDECAR_TAG="${SIDECAR_TAG}" \
SHELL_NAME="$(basename "$SHELL")" \
PROJROOT="$(git rev-parse --show-toplevel)" \
GIT_BRANCH="$(git rev-parse --abbrev-ref HEAD |sed 's@/@_@g')" \
  go run "${proj_root_dir}/hack/envexpand.go" \
  ${proj_root_dir}/hack/k8s/${TEMPLATE_NAME}.template.yaml \
  > "$RES_DEF"

if kubectl -n $NS get deployment "${DEPLOYMENT_NAME}" &> /dev/null; then
	kubectl -n $NS delete deployment "${DEPLOYMENT_NAME}"
fi

dbg_print '### creating from templated yaml'
dbg_print '----'
dbg_print "$(cat ${RES_DEF})"
dbg_print '----'

kubectl -n $NS create -f ${RES_DEF}
