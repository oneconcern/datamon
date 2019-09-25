#! /bin/zsh

setopt ERR_EXIT

SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"

OUTPUT_LABEL=pg-coord-example
IS_INIT=false
IGNORE_VERSION_MISMATCH=false

proj_root_dir="$(dirname "$(dirname "$SCRIPT_DIR")")"

pull_policy=Always

typeset -i SOME_CONST

while getopts ioc: opt; do
    case $opt in
        (i)
            OUTPUT_LABEL=pg-coord-example-input
            IS_INIT=true
            IGNORE_VERSION_MISMATCH=true
            ;;
        (o)
            # local deploy
            pull_policy=IfNotPresent
            ;;
        (c)
            SOME_CONST="$OPTARG"
            ;;
        (\?)
            print Bad option, aborting.
            exit 1
            ;;
    esac
done
(( OPTIND > 1 )) && shift $(( OPTIND - 1 ))

if [[ -z $GOOGLE_APPLICATION_CREDENTIALS ]]; then
	echo 'GOOGLE_APPLICATION_CREDENTIALS env variable not set' 1>&2
	exit 1
fi

if kubectl get secret google-application-credentials &> /dev/null; then
	kubectl delete secret google-application-credentials
fi

# https://cloud.google.com/kubernetes-engine/docs/tutorials/authenticating-to-cloud-platform#step_4_import_credentials_as_a_secret
kubectl create secret generic \
	google-application-credentials \
	--from-file=google-application-credentials.json=$GOOGLE_APPLICATION_CREDENTIALS

RES_DEF="$proj_root_dir"/hack/k8s/gen/example-coord-pg.yaml

IGNORE_VERSION_MISMATCH=$IGNORE_VERSION_MISMATCH \
IS_INIT=$IS_INIT \
PULL_POLICY=$pull_policy \
OUTPUT_LABEL=$OUTPUT_LABEL \
SOME_CONST=$SOME_CONST \
	"$proj_root_dir"/hack/envexpand "$proj_root_dir"/hack/k8s/example-coord-pg.template.yaml > "$RES_DEF"

if kubectl get deployment datamon-coord-pg-demo &> /dev/null; then
	kubectl delete -f "$RES_DEF"
fi

kubectl create -f "$RES_DEF"
