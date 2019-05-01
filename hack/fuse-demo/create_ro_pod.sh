#! /bin/zsh

SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"

proj_root_dir="$(dirname "$(dirname "$SCRIPT_DIR")")"

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

RES_DEF="$proj_root_dir"/hack/k8s/gen/example-ro.yaml

SHELL_NAME="$(basename "$SHELL")" \
	PROJROOT="$(git rev-parse --show-toplevel)" \
	GIT_BRANCH="$(git rev-parse --abbrev-ref HEAD |sed 's@/@_@g')" \
	"$proj_root_dir"/hack/envexpand "$proj_root_dir"/hack/k8s/example-ro.template.yaml > "$RES_DEF"

if kubectl get deployment datamon-ro-demo &> /dev/null; then
	kubectl delete -f "$RES_DEF"
fi

kubectl create -f "$RES_DEF"
