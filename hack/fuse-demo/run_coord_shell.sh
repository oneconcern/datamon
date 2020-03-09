#! /bin/zsh

setopt ERR_EXIT

dbg_print() {
    local COL_GREEN
    local COL_RESET
    COL_YELLOW=$(tput -Txterm setaf 3)
    COL_RESET=$(tput -Txterm sgr0)
    echo ${COL_YELLOW}
    print -- $1
    echo ${COL_RESET}
}



container_name=demo-app
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


pod_name=$(kubectl -n $NS get pods -l app=datamon-coord-demo | grep Running | sed 's/ .*//')

if [[ -z $pod_name ]]; then
	echo 'coord demo pod not found' 1>&2
	exit 1
fi

# todo: pass $RES_DEF as an optional parameter and
#   use shell yaml query tool to get container names,
#   and log $RES_DEF where <unspecified>
dbg_print 'starting "'"$container_name"'" from <unspecified> yaml'

kubectl -n $NS exec -it "$pod_name" \
        -c "$container_name" \
        -- "/bin/zsh"
