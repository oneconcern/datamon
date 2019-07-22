#! /bin/zsh

## create pod

SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"

proj_root_dir="$(dirname "$SCRIPT_DIR")"

RES_DEF="$proj_root_dir"/hack/k8s/circleci-builder.yaml

if [[ ! -e $RES_DEF ]]; then
    echo "'$RES_DEF' not found" 1>&2
    exit 1
fi

if kubectl get deployment datamon-circleci-builder &> /dev/null; then
	kubectl delete -f "$RES_DEF"
fi

kubectl create -f "$RES_DEF"

## wait on start

STARTUP_POLL_INTERVAL=1

pod_name=

echo "waiting on pod start"

while [[ -z $pod_name ]]; do
    sleep "$STARTUP_POLL_INTERVAL"
    pod_name=$(kubectl get pods -l app=datamon-circleci-builder | grep Running | sed 's/ .*//')
done

## exec shell

container_name=shell

kubectl exec -it "$pod_name" \
        -c "$container_name" \
        -- "/bin/zsh"
