#! /bin/zsh

STARTUP_POLL_INTERVAL=1

pod_name=

echo "waiting on pod start"

while [[ -z $pod_name ]]; do
    sleep "$STARTUP_POLL_INTERVAL"
    pod_name=$(kubectl get pods -l app=datamon-coord-demo | grep Running | sed 's/ .*//')
done

echo "pod started, following logs of $pod_name"

# kubectl logs -f -lapp=datamon-coord-demo --all-containers=true

LOG_FILE=/tmp/datamon-coord-demo.log

if [[ -e $LOG_FILE ]]; then
    print "removing extant log file $LOG_FILE"
    rm $LOG_FILE
fi

kubectl logs "$pod_name" -f --all-containers=true |tee $LOG_FILE &
log_pid=$!

LOG_POLL_INTERVAL=3

WRAP_APPLICATION_DONE=
WRAP_DATAMON_DONE=

while [[ -z $WRAP_APPLICATION_DONE || -z $WRAP_DATAMON_DONE ]]; do
    if cat $LOG_FILE | grep -q '^wrap_application sleeping indefinitely'; then
        WRAP_APPLICATION_DONE=true
    fi
    if cat $LOG_FILE | grep -q '^wrap_datamon sleeping indefinitely'; then
        WRAP_DATAMON_DONE=true
    fi
    sleep $LOG_POLL_INTERVAL
done
