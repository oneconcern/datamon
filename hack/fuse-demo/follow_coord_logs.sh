#! /bin/sh

STARTUP_POLL_INTERVAL=1

pod_name=

echo "waiting on pod start"

while [[ -z $pod_name ]]; do
    sleep "$STARTUP_POLL_INTERVAL"
    pod_name=$(kubectl get pods -l app=datamon-coord-demo | grep Running | sed 's/ .*//')
done

echo "pod started, following logs of $pod_name"

# kubectl logs -f -lapp=datamon-coord-demo --all-containers=true

kubectl logs "$pod_name" -f --all-containers=true
