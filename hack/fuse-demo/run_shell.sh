#! /bin/zsh

STARTUP_POLL_INTERVAL=1

container_name=demo-shell

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

pod_name=$(kubectl get pods -l app=datamon-ro-demo | grep Running | sed 's/ .*//')

echo "waiting on pod start"

while [[ -z $pod_name ]]; do
    sleep "$STARTUP_POLL_INTERVAL"
    pod_name=$(kubectl get pods -l app=datamon-ro-demo | grep Running | sed 's/ .*//')
done

kubectl exec -it "$pod_name" \
        -c "$container_name" \
        -- "/bin/zsh"
