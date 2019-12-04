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

CLUSTER_GAC_SECRET_NAME=google-application-credentials

dbg_print '### creating secret '"${CLUSTER_GAC_SECRET_NAME}"' in cluster'
dbg_print '### from file "'"${GOOGLE_APPLICATION_CREDENTIALS}"'"'
if [[ -z $GOOGLE_APPLICATION_CREDENTIALS ]]; then
	  echo 'GOOGLE_APPLICATION_CREDENTIALS env variable not set' 1>&2
	  exit 1
fi

if kubectl get secret ${CLUSTER_GAC_SECRET_NAME} &> /dev/null; then
    dbg_print '##### named secret exists so deleting'
	  kubectl delete secret ${CLUSTER_GAC_SECRET_NAME}
fi
dbg_print '##### creating according to'
dbg_print 'https://cloud.google.com/kubernetes-engine/docs/tutorials/authenticating-to-cloud-platform#step_4_import_credentials_as_a_secret'
kubectl create secret generic \
	${CLUSTER_GAC_SECRET_NAME} \
	--from-file=google-application-credentials.json=${GOOGLE_APPLICATION_CREDENTIALS}

##

TEMPLATE_NAME='example-coord'
if $shell_only; then
    TEMPLATE_NAME='example-coord_shell-only'
fi
RES_DEF=${proj_root_dir}/hack/k8s/gen/${TEMPLATE_NAME}.yaml
dbg_print '### templating k8s api server yaml for kubectl cmd to '"${RES_DEF}"
PULL_POLICY=$pull_policy \
SHELL_NAME="$(basename "$SHELL")" \
	PROJROOT="$(git rev-parse --show-toplevel)" \
	GIT_BRANCH="$(git rev-parse --abbrev-ref HEAD |sed 's@/@_@g')" \
	${proj_root_dir}/hack/envexpand \
  ${proj_root_dir}/hack/k8s/${TEMPLATE_NAME}.template.yaml \
  > "$RES_DEF"

if kubectl get deployment datamon-coord-demo &> /dev/null; then
	kubectl delete -f "$RES_DEF"
fi

dbg_print '### creating from templated yaml'
dbg_print '----'
dbg_print "$(cat ${RES_DEF})"
dbg_print '----'

kubectl create -f ${RES_DEF}
