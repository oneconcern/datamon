#! /bin/zsh

SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"
PROJ_ROOT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

TIMESTAMP_HUMAN_READABLE=$(date '+%Y_%b_%d_%s' | tr '[:upper:]' '[:lower:]')

docker_img_tag=latest
unlinkable_list="/filestore/datamover-backup-wd/$TIMESTAMP_HUMAN_READABLE"

is_test_debug=true
pvc_name=depmon-pvc

create_job=true

while getopts rft:x opt; do
    case $opt in
        (r)
            print 'running in real mode'
            pvc_name=pvc1
            is_test_debug=false
            ;;
        (f)
            create_job=false
            ;;
        (t)
            docker_img_tag="$OPTARG"
            ;;
        (x)
            print 'will delete files'
            unlinkable_list=''
            ;;
        (\?)
            print Bad option, aborting.
            exit 1
            ;;
    esac
done
(( OPTIND > 1 )) && shift $(( OPTIND - 1 ))

# time to interrupt in case of dangerous params
sleep 5

# could verify that tag exist aot

##

job_name=datamon-backup-job

k8s_yaml_name=datamover_backup
res_def="${PROJ_ROOT_DIR}"/hack/k8s/gen/${k8s_yaml_name}.yaml

PVC_NAME=$pvc_name \
IS_TEST_DEBUG=$is_test_debug \
UNLINKABLE_LIST=$unlinkable_list \
DOCKER_IMG_TAG=$docker_img_tag \
    "${PROJ_ROOT_DIR}"/hack/envexpand \
    "${PROJ_ROOT_DIR}"/hack/k8s/${k8s_yaml_name}.template.yaml \
    > "$res_def"

if $create_job; then
    if kubectl get job ${job_name} &> /dev/null; then
	      kubectl delete -f "$res_def"
    fi
    kubectl create -f "$res_def"
fi

STARTUP_POLL_INTERVAL=1
print 'waiting on job start'
job_started=false
while ! $job_started; do
    sleep "$STARTUP_POLL_INTERVAL"
    if kubectl get job ${job_name} &> /dev/null; then
        job_started=true
    fi
done
kubectl logs -f job.batch/${job_name}
