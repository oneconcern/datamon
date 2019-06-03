#! /bin/zsh

pod_name=$(kubectl get pods -l app=datamon-coord-demo | grep Running | sed 's/ .*//')

if [[ -z $pod_name ]]; then
	echo 'coord demo pod not found' 1>&2
	exit 1
fi

container_name=demo-app

if [ -z "$1" ]; then
    container_name='datamon-sidecar'
fi

kubectl exec -it "$pod_name" \
        -c "$container_name" \
        -- "/bin/bash"
