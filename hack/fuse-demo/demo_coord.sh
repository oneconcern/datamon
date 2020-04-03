#! /bin/zsh

## demonstrate a fuse read-only filesystem

setopt ERR_EXIT

dbg_print() {
    local COL_GREEN
    local COL_RESET
    COL_GREEN=$(tput -Txterm setaf 2)
    COL_RESET=$(tput -Txterm sgr0)
    echo ${COL_GREEN}
    print -- $1
    echo ${COL_RESET}
}

##

typeset -A known_k8s_tags_to_ctxs
known_k8s_tags_to_ctxs[local]=docker-desktop
known_k8s_tags_to_ctxs[remote]=gke_onec-co_us-west2-c_onec-dev
known_k8s_ctxs=(${(v)known_k8s_tags_to_ctxs})

k8s_ctx_opt=remote
build_sidecar_base=false
#
build_demo_app=false
build_local=false
#
shell_only=false

while getopts osbS opt; do
    case $opt in
        (o)
            # local deploy
            k8s_ctx_opt=local
            ;;
        (s)
            build_demo_app=true
            build_local=true
            ;;
        (b)
            build_sidecar_base=true
            ;;
        (S)
            # local deploy
            shell_only=true
            ;;
        (\?)
            print Bad option, aborting.
            exit 1
            ;;
    esac
done
(( OPTIND > 1 )) && shift $(( OPTIND - 1 ))

dbg_print 'demo_coord.sh getopts end'

dbg_print '### building images'

run_docker_make_cmd() {
    local docker_make_target=$1
    local log_file="/tmp/datamon-coord-demo-make-docker_${docker_make_target}.log"
    dbg_print "log at ${log_file}"
    2>&1 make $docker_make_target | \
        tee $log_file | \
        grep '^#'
}


if [[ $build_sidecar_base == "true" ]] ; then
    dbg_print '##### building sidecar image'
    run_docker_make_cmd build-and-push-fuse-sidecar
else
    dbg_print '##### skipping sidecar image build'
fi
if [[ $build_demo_app == "true" ]] ; then
    dbg_print '##### building coord app image'
    run_docker_make_cmd fuse-demo-coord-build-app
else
    dbg_print '##### skipping coord app image build'
fi
if [[  $build_local == "true" ]] ; then
    dbg_print '##### building datamon local binary'
    dbg_print 'in order to verify uploaded bundle locally'
    make build-datamon-local
    sudo install out/datamon /usr/bin
else
    dbg_print '##### skipping local datamon build'
fi

dbg_print '### starting demo in k8s'

dbg_print '##### creating coordination pod'
typeset -a create_coord_pod_opts
if [[ $shell_only == "true" ]] ; then
    create_coord_pod_opts=('-S' $create_coord_pod_opts)
fi
if [[ $k8s_ctx_opt = 'local' ]]; then
    dbg_print 'creating coordination pod for local docker-desktop'
    create_coord_pod_opts=('-o' $create_coord_pod_opts)
fi
./hack/fuse-demo/create_coord_pod.sh $create_coord_pod_opts

if [[ $shell_only == "true" ]] ; then
    dbg_print '### running coordination shell'
    dbg_print '##### dumb timeout on start'
    for i in $(seq 35); do
        echo -n '.'
        sleep 1
    done
    dbg_print '##### attempting to drop into shell'
    dbg_print 'run '
    dbg_print '> ./hack/fuse-demo/run_coord_shell.sh -s'
    dbg_print 'to start sidecar shell'
    ./hack/fuse-demo/run_coord_shell.sh
    exit 0
fi

POLL_INTERVAL=1
NS=datamon-ci
pod_name=
SIDECAR_TAG=$(go run ./hack/release_tag.go)

dbg_print 'waiting on pod start'
typeset -i COUNT
COUNT=0
dbg_print "##### pod with api server metadata app=datamon-coord-fuse-demo-pods,instance=${SIDECAR_TAG}"
while [[ -z $pod_name ]]; do
    sleep "$POLL_INTERVAL"

    if ! k=$(kubectl -n "${NS}" get pods -l app=datamon-coord-fuse-demo,instance="${SIDECAR_TAG}" --output custom-columns=NAME:.metadata.name,STATUS:.status.phase) ; then
      # sometimes, we lose connectivity from the circleCI container: handle failure and retry a couple times
      error_print "cannot fetch pod logs. Retrying..."
      COUNT=$((COUNT+1))
      if [[ "${COUNT}" -gt 10 ]] ; then
        error_print "cannot fetch pod logs. Giving up after ${COUNT} attempts."
        exit 1
      fi
      continue
    fi
    pod_name=$(echo "${k}" | grep Running | cut -d' ' -f1) || true
    check=$(echo "${k}"|grep -iE '(BackOff)|(Error)') || true
    if [[ -n "${check}" ]] ; then
      dbg_print "pod won't start. Exiting..."
      dbg_print "${k}"
      exit 1
    fi
done

dbg_print '##### placed timestamp aot for verification'
pod_time="$(kubectl -n "${NS}" get pod "${pod_name}" --template '{{ range .status.conditions }}{{ printf "%v\n" .lastTransitionTime }}{{ end }}'|sort|head -1)"
timestamp="$(go run ./hack/fuse-demo/parse_timestamp.go "${pod_time}")"
echo "${timestamp}" > /tmp/datamon_fuse_demo_coord_start_timestamp
echo "measured pod local time: ${pod_time} => ${timestamp}"

LOG_FILE=/tmp/datamon-coord-demo.log

dbg_print "pod started, following logs from $pod_name"
rm -f "${LOG_FILE}" && touch "${LOG_FILE}"
(sleep 600 && echo "timed out on waiting" && kill -15 $$ || exit 1) & # stops waiting after 10m
(kubectl -n "${NS}" logs -f --all-containers=true "${pod_name}"|tee -a "${LOG_FILE}" || true) &
# now wait for the demo to complete
(tail -f "${LOG_FILE}" || true)|while read -r line ; do
  if [[ "${line}" =~ "application exited with non-zero-status" ]] ; then
    dbg_print "error detecting in a container"
    exit 1
  fi
  if [[ "${line}" =~ "wrap_application sleeping indefinitely" ]] ; then
    break
  fi
done
dbg_print "success: wrapper ended successfully"

dbg_print '##### following coord logs ended'
dbg_print '### verifying locally'
./hack/fuse-demo/verify_coord_bundle.sh
