#! /bin/zsh

## demonstrate a fuse read-only filesystem

# on the utility shell scripts v. makefile targets by use-case
# https://stackoverflow.com/a/45003119

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
build_demo_sidecar=true
build_demo_app=true
build_local=true
#
shell_only=false

while getopts osbS opt; do
    case $opt in
        (o)
            # local deploy
            k8s_ctx_opt=local
            ;;
        (s)
            build_demo_sidecar=false
            build_demo_app=false
            build_local=false
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

k8s_ctx=
if [[ ${known_k8s_ctxs[(ie)$k8s_ctx_opt]} -le ${#known_k8s_ctxs} ]]; then
    k8s_ctx=$k8s_ctx_opt
else
    k8s_ctx=${known_k8s_tags_to_ctxs[$k8s_ctx_opt]}
fi
if [[ -z $k8s_ctx ]]; then
    print 'kubernetes context not set' 1>&2
    exit 1
fi
if [[ -z $GCLOUD_SERVICE_KEY ]]; then
    # not in ci
    kubectx $k8s_ctx
fi

##

dbg_print '### using golang api to write serialized input file'


##

dbg_print '### building images'

run_docker_make_cmd() {
    local docker_make_target=$1
    local log_file="/tmp/datamon-coord-demo-make-docker_${docker_make_target}.log"
    dbg_print "log at ${log_file}"
    2>&1 make $docker_make_target | \
        tee $log_file | \
        grep '^#'
}


if $build_sidecar_base; then
    dbg_print '##### building sidecar image'
    run_docker_make_cmd build-and-push-fuse-sidecar
else
    dbg_print '##### skipping sidecar image build'
fi
dbg_print '##### building coord app image'
if $build_demo_app; then
    run_docker_make_cmd fuse-demo-coord-build-app
fi
dbg_print '##### building demo sidecar image'
if $build_demo_sidecar; then
    run_docker_make_cmd fuse-demo-coord-build-datamon
fi
dbg_print '##### building datamon local image'
dbg_print 'in order to verify uploaded bundle locally'
if $build_local; then
    make build-datamon-local
fi

dbg_print '### starting demo in k8s'

dbg_print '##### placed timestamp aot for verification'
date '+%s' > /tmp/datamon_fuse_demo_coord_start_timestamp

dbg_print '##### creating coordiation pod'
typeset -a create_coord_pod_opts
if $shell_only; then
    create_coord_pod_opts=('-S' $create_coord_pod_opts)
fi
if [[ $k8s_ctx_opt = 'local' ]]; then
    dbg_print 'creating coordiation pod for local docker-desktop'
    create_coord_pod_opts=('-o' $create_coord_pod_opts)
fi
./hack/fuse-demo/create_coord_pod.sh $create_coord_pod_opts

if ! $shell_only; then
    dbg_print '##### following coodination pod logs until demo finished'
    ./hack/fuse-demo/follow_coord_logs.sh
    dbg_print '##### following coord logs ended'
    dbg_print '### verifying locally'
    ./hack/fuse-demo/verify_coord_bundle.sh
else
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
fi
