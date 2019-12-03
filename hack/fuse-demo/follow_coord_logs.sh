#! /bin/zsh


setopt ERR_EXIT
setopt PIPE_FAIL

dbg_print() {
    local COL_YELLOW
    local COL_RESET
    COL_YELLOW=$(tput -Txterm setaf 3)
    COL_RESET=$(tput -Txterm sgr0)
    echo ${COL_YELLOW}
    print -- $1
    echo ${COL_RESET}
}

#

STARTUP_POLL_INTERVAL=1

pod_name=

dbg_print 'waiting on pod start'

typeset -i last_pods_poll

if [[ -n $last_pods_poll ]]; then
    dbg_print '##### last_pods_poll set before start'
fi

while [[ -z $pod_name ]] || \
          [[ ! $last_pods_poll -eq 0 ]]; do
    for x in $(seq 1 3); do
        echo -n '.'
        sleep 1
    done
    echo

    dbg_print '##### pods with api server metadata app=datamon-coord-demo-pods'
    dbg_print '===='
    dbg_print "$(kubectl get pods -l app=datamon-coord-demo)"
    dbg_print '----'
    dbg_print "$(kubectl get pods -l app=datamon-coord-demo 2>&1 | grep Running)"
    dbg_print '----'
    dbg_print "$(kubectl get pods -l app=datamon-coord-demo | grep Running | sed 's/ .*//' )"
    dbg_print '===='

    unsetopt ERR_EXIT
    unsetopt PIPE_FAIL
    pod_name=$(2>&1 kubectl get pods -l app=datamon-coord-demo | grep Running | sed 's/ .*//')
    last_pods_poll=$?
    setopt ERR_EXIT
    setopt PIPE_FAIL

    if [[ ! $last_pods_poll -eq 0 ]]; then
        dbg_print "error on getting datamon-coord-demo"
        dbg_print "out of k8s api server metadata"
        print --
        print -- $pod_name 1>&2
        print --
    else
        dbg_print "pods poll finished with nominal status $last_pods_poll"
        print -- $pod_name
        dbg_print "> ${pod_name}"
    fi
    sleep "$STARTUP_POLL_INTERVAL"

    if [[ ! $last_pods_poll -eq 0 ]]; then
        dbg_print '#### cond aff'
    else
        dbg_print '#### cond neg'
        dbg_print "##### pod_name ${pod_name}"
        if [[ -z $pod_name ]]; then
            dbg_print '##### empty pod name'
        fi
    fi

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
