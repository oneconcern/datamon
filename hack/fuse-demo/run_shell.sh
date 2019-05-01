#! /bin/zsh

pod_name=$(kubectl get pods -l app=datamon-ro-demo | grep Running | sed 's/ .*//')

if [[ -z $pod_name ]]; then
	echo 'fuse demo pod not found' 1>&2
	exit 1
fi

# datamon-sidecar
# demo-shell
kubectl exec -it "$pod_name" -c demo-shell -- "/bin/bash"
