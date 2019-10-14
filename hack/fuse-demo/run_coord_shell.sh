#! /bin/zsh

container_name=demo-app

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

typeset pod_name

print -- "waiting on pod start"

while [[ -z $pod_name ]]; do
    sleep 1
    pod_name=$(kubectl get pods -l app=datamon-coord-demo | \
                   grep Running | sed 's/ .*//')
done

kubectl exec -it $pod_name \
        -c $container_name \
        -- "/bin/zsh"
