#! /bin/zsh

SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"
PROJ_ROOT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

TIMESTAMP=$(date +%y%m%d%H%M%S)

### be sure to run in dev env,
# `kubens dev`,
# in order to access NFS Share PVC

pvc_mnt_path=/mnt/shared

# used
# '/mnt/shared/datamon/move_output/july26'
# in test
bkup_path_opt=

timestamp_filter_before=090725000000

filelist_dir=datamover-lists-${TIMESTAMP}

while getopts p:t:f: opt; do
    case $opt in
        (p)
            bkup_path_opt="$OPTARG"
            ;;
        (t)
            timestamp_filter_before="$OPTARG"
            ;;
        (f)
            filelist_dir="$OPTARG"
            ;;
        (\?)
            print Bad option, aborting.
            exit 1
            ;;
    esac
done
(( OPTIND > 1 )) && shift $(( OPTIND - 1 ))

if [[ -z $bkup_path_opt ]]; then
    print "backup path not specified.  specify with -p option." 1>&2
    exit 1
fi

if ! print $bkup_path_opt | grep -q '^/'; then
    bkup_path_opt=${pvc_mnt_path}/${bkup_path_opt}
fi
if ! print $filelist_dir | grep -q '^/'; then
    filelist_dir=${pvc_mnt_path}/${filelist_dir}
fi

if [[ -z $GOOGLE_APPLICATION_CREDENTIALS ]]; then
	echo 'GOOGLE_APPLICATION_CREDENTIALS env variable not set' 1>&2
	exit 1
fi

RES_DEF="${PROJ_ROOT_DIR}"/hack/k8s/gen/datamover_job.yaml

# pvc1 is the NFS PVC name

TIMESTAMP_FILTER_BEFORE=$timestamp_filter_before \
PVC_MNT_PATH=$pvc_mnt_path \
FILELIST_DIR=$filelist_dir \
PVC_NAME=depmon-pvc \
BKUP_PATH=$bkup_path_opt \
         "${PROJ_ROOT_DIR}"/hack/envexpand \
         "${PROJ_ROOT_DIR}"/hack/k8s/datamover_job.template.yaml \
         > "$RES_DEF"

##

if kubectl get secret google-application-credentials &> /dev/null; then
	kubectl delete secret google-application-credentials
fi

# https://cloud.google.com/kubernetes-engine/docs/tutorials/authenticating-to-cloud-platform#step_4_import_credentials_as_a_secret
kubectl create secret generic \
	google-application-credentials \
	--from-file=google-application-credentials.json=$GOOGLE_APPLICATION_CREDENTIALS

kubectl delete job.batch/datamon-datamover-job

kubectl create -f "$RES_DEF"
