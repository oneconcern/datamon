#! /bin/zsh

container_name=datamon-bin

STARTUP_POLL_INTERVAL=1

pod_name=

echo "waiting on pod start"

while [[ -z $pod_name ]]; do
    sleep "$STARTUP_POLL_INTERVAL"
    pod_name=$(kubectl get pods -l app=datamon-datamover-job | grep Running | sed 's/ .*//')
done

kubectl exec -it "$pod_name" \
        -c "$container_name" \
        -- "/bin/zsh"
