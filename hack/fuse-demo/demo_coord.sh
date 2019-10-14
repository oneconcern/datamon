#! /bin/zsh

## demonstrate a fuse read-only filesystem

# on the utility shell scripts v. makefile targets by use-case
# https://stackoverflow.com/a/45003119

setopt ERR_EXIT

typeset -A known_k8s_tags_to_ctxs
known_k8s_tags_to_ctxs[local]=docker-desktop
known_k8s_tags_to_ctxs[remote]=gke_onec-co_us-west2-c_onec-dev
known_k8s_ctxs=(${(v)known_k8s_tags_to_ctxs})

k8s_ctx_opt=remote
build_datamon_base=false
build_sidecar=true
build_demo_sidecar=true
build_demo_app=true
build_local=true

while getopts os opt; do
    case $opt in
        (b)
            build_datamon_base=true
            ;;
        (o)
            # local deploy
            k8s_ctx_opt=local
            ;;
        (s)
            build_sidecar=false
            build_demo_sidecar=false
            build_demo_app=false
            build_local=false
            ;;
        (\?)
            print Bad option, aborting.
            exit 1
            ;;
    esac
done
(( OPTIND > 1 )) && shift $(( OPTIND - 1 ))

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
kubectx $k8s_ctx

##

if $build_datamon_base; then
    make build-datamon-binaries
fi
if $build_sidecar; then
    make build-and-push-fuse-sidecar-img
fi
if $build_demo_app; then
    make fuse-demo-coord-build-app
fi
if $build_demo_sidecar; then
    make fuse-demo-coord-build-datamon
fi
if $build_local; then
    make build-datamon-local
fi

date '+%s' > /tmp/datamon_fuse_demo_coord_start_timestamp


if [[ $k8s_ctx_opt = 'local' ]]; then
    ./hack/fuse-demo/create_coord_pod.sh -o
    else
        ./hack/fuse-demo/create_coord_pod.sh
fi
./hack/fuse-demo/follow_coord_logs.sh
./hack/fuse-demo/verify_coord_bundle.sh
