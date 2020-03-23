#! /bin/zsh

container_name=demo-shell
NS=datamon-ci
while getopts s opt; do
    case $opt in
        (s)
            container_name='datamon-sidecar'
            ;;
        (\?)
            print Bad option, aborting.
            exit 1
            ;;
    esac
done
(( OPTIND > 1 )) && shift $(( OPTIND - 1 ))

STARTUP_POLL_INTERVAL=1
typeset pod_name

print -- "waiting on pod start"

while [[ -z $pod_name ]]; do
    sleep "$STARTUP_POLL_INTERVAL"
    pod_name=$(kubectl -n $NS get pods -l app=datamon-ro-demo | grep Running | sed 's/ .*//')
done

kubectl -n $NS exec -it "$pod_name" \
        -c "$container_name" \
        -- "/bin/zsh"
