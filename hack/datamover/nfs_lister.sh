#! /bin/zsh

SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"
PROJ_ROOT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

list_out=

pvc_mnt_path=/mnt/shared
pvc_name=depmon-pvc

bkup_path_opt=

while getopts n:p:o: opt; do
    case $opt in
        (o)
            list_out="$OPTARG"
            ;;
        (p)
            bkup_path_opt="$OPTARG"
            ;;
        (n)
            pvc_name="$OPTARG"
            ;;
        (\?)
            print Bad option, aborting.
            exit 1
            ;;
    esac
done
(( OPTIND > 1 )) && shift $(( OPTIND - 1 ))

if ! print $bkup_path_opt | grep -q '^/'; then
    bkup_path_opt=${pvc_mnt_path}/${bkup_path_opt}
fi


template_name=datamover_nfs_lister
RES_DEF="${PROJ_ROOT_DIR}"/hack/k8s/gen/${template_name}.yaml

PVC_MNT_PATH=$pvc_mnt_path \
PVC_NAME=$pvc_name \
BKUP_PATH=$bkup_path_opt \
         "${PROJ_ROOT_DIR}"/hack/envexpand \
         "${PROJ_ROOT_DIR}"/hack/k8s/${template_name}.template.yaml \
         > "$RES_DEF"

kubectl delete job.batch/datamon-datamover-lister

kubectl create -f "$RES_DEF"

AWAIT_COMPLETION_POLL_INTERVAL=1

num_completions=0
while [[ $num_completions -eq 0 ]]; do
    sleep $AWAIT_COMPLETION_POLL_INTERVAL
    num_completions=$(kubectl get job datamon-datamover-lister | \
                          tail -1 | \
                          tr -s ' ' | \
                          cut -d ' ' -f 2 | \
                          cut -d'/' -f 1)
done

if [[ -z $list_out ]]; then
    kubectl logs jobs.batch/datamon-datamover-lister | sed 's/\(.*\)/item:\1/'
else
    kubectl logs jobs.batch/datamon-datamover-lister &> $list_out
fi
