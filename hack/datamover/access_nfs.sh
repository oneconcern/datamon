#! /bin/zsh

SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"
PROJ_ROOT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

POD_START_POLL_INTERVAL=1

typeset -A known_nfs_pvc_tags_to_names
known_nfs_pvc_tags_to_names[argo]=pvc1
known_nfs_pvc_tags_to_names[test]=depmon-pvc
known_nfs_pvc_names=(${(v)known_nfs_pvc_tags_to_names})

pvc_name_opt=argo
create_pod=false
docker_img_tag=latest

while getopts cp:t: opt; do
    case $opt in
        (c)
            create_pod=true
            ;;
        (p)
            pvc_name_opt="$OPTARG"
            ;;
        (t)
            docker_img_tag="$OPTARG"
            ;;
        (\?)
            print Bad option, aborting.
            exit 1
            ;;
    esac
done
(( OPTIND > 1 )) && shift $(( OPTIND - 1 ))


pvc_name=
if [[ ${known_nfs_pvc_names[(ie)$pvc_name_opt]} -le ${#known_nfs_pvc_names} ]]; then
    pvc_name=$pvc_name_opt
else
    pvc_name=${known_nfs_pvc_tags_to_names[$pvc_name_opt]}
fi
if [[ -z $pvc_name ]]; then
    print 'pvc name not set' 1>&2
    exit 1
fi

deployment_name=datamon-datamover-nfs-access

if $create_pod; then
    k8s_yaml_name=datamover_nfs_access
    res_def="${PROJ_ROOT_DIR}"/hack/k8s/gen/${k8s_yaml_name}.yaml
    DOCKER_IMG_TAG=$docker_img_tag \
    PVC_NAME=$pvc_name \
    PVC_MNT_PATH=/filestore \
            "${PROJ_ROOT_DIR}"/hack/envexpand \
            "${PROJ_ROOT_DIR}"/hack/k8s/${k8s_yaml_name}.template.yaml \
            > "$res_def"
    if kubectl get deployment ${deployment_name} &> /dev/null; then
	      kubectl delete -f "$res_def"
    fi
    kubectl create -f "$res_def"
else
    print 'skipping pod creation'
fi

if ! &>/dev/null kubectl get deployment $deployment_name; then
    print "deployment_name $deployment_name not found" 1>&2
    print 'try using the -c option to create a pod'
    exit 1
fi

print 'waiting on pod start'
pod_name=
while [[ -z $pod_name ]]; do
    sleep "$POD_START_POLL_INTERVAL"
    pod_name=$(kubectl get pods -l app=datamon-datamover-nfs-access | grep Running | sed 's/ .*//')
done

kubectl exec -it "$pod_name" \
        -c datamon-bin \
        -- "/bin/zsh"
