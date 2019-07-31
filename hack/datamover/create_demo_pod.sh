#! /bin/zsh

SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"
PROJ_ROOT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

### be sure to run in dev env,
# `kubens dev`,
# in order to access NFS Share PVC

# used
# '/mnt/shared/datamon/move_output/july26'
# in test
BKUP_PATH_OPT=

tot_size_tb=1
num_files=100000

write_files_only=false

while getopts os:n:p: opt; do
    case $opt in
        (o)
            write_files_only=true
            ;;
        (s)
            tot_size_tb="$OPTARG"
            ;;
        (n)
            num_files="$OPTARG"
            ;;
        (p)
            BKUP_PATH_OPT="$OPTARG"
            ;;
        (\?)
            print Bad option, aborting.
            exit 1
            ;;
    esac
done
(( OPTIND > 1 )) && shift $(( OPTIND - 1 ))

# if [[ -z $BKUP_PATH_OPT ]]; then
#     print "backup path not specified.  specify with -p option." 1>&2
#     exit 1
# fi

RES_DEF="${PROJ_ROOT_DIR}"/hack/k8s/gen/datamover.yaml

WRITE_FILES_ONLY=$write_files_only \
PVC_NAME=depmon-pvc \
PVC_MNT_PATH=/mnt/shared \
TOT_SIZE_TB=$tot_size_tb \
NUM_FILES=$num_files \
BKUP_PATH=$BKUP_PATH_OPT \
  "${PROJ_ROOT_DIR}"/hack/envexpand \
    "${PROJ_ROOT_DIR}"/hack/k8s/datamover.template.yaml \
    > "$RES_DEF"

if kubectl get deployment datamon-datamover-test &> /dev/null; then
	kubectl delete -f "$RES_DEF"
fi

kubectl create -f "$RES_DEF"
