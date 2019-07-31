#! /bin/zsh

container_name=datamon-bin

STARTUP_POLL_INTERVAL=1

pod_name=

echo "waiting on job completion"

while [[ -z $pod_name ]]; do
    sleep "$STARTUP_POLL_INTERVAL"
    pod_name=$(kubectl get pods -l app=datamon-datamover-job | grep Completed | sed 's/ .*//')
done


echo "pod started, following logs of $pod_name"

kubectl logs "$pod_name" -c datamon-bin |humanlog
