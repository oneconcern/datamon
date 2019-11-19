#! /bin/zsh

## demonstrate pg batteries

setopt ERR_EXIT

DATAMON_EXEC=./cmd/datamon/datamon

build_datamon_base=false
build_sidecar=true
build_demo_app=true
init_demo_bundle=false

# roundtrip thru sidecar.
# change this variable to demonstrate dynamic upload contents
typeset -i SOME_CONST

while getopts dbsc: opt; do
    case $opt in
        (d)
            init_demo_bundle=true
            ;;
        (b)
            build_datamon_base=true
            ;;
        (s)
            build_sidecar_base=false
            build_sidecar=false
            build_demo_app=false
            ;;
        (c)
            SOME_CONST="$OPTARG"
            ;;
        (\?)
            print Bad option, aborting.
            exit 1
            ;;
    esac
done
(( OPTIND > 1 )) && shift $(( OPTIND - 1 ))

if $build_datamon_base; then
    make build-datamon-binaries
fi
if $build_sidecar; then
    make build-and-push-pg-sidecar
fi
if $build_demo_app; then
    make pg-demo-coord-build-app
fi

make build-datamon-local

##

date '+%s' > /tmp/datamon_pg_demo_coord_start_timestamp

params_create_pod=(-c $SOME_CONST)
if $init_demo_bundle; then
    params_create_pod=($params_create_pod -i)
fi
print -- "creating coord pod with params '${params_create_pod}'"
./hack/fuse-demo/create_coord_pg_pod.sh $params_create_pod

##### follow_coord_logs.sh dupe

STARTUP_POLL_INTERVAL=1


pod_name=

echo "waiting on pod start"

while [[ -z $pod_name ]]; do
    sleep "$STARTUP_POLL_INTERVAL"
    pod_name=$(kubectl get pods -l app=datamon-coord-pg-demo | grep Running | sed 's/ .*//')
done

echo "pod started, following logs of $pod_name"

LOG_FILE=/tmp/datamon-coord-pg-demo.log
if [[ -e $LOG_FILE ]]; then
    print "removing extant log file $LOG_FILE"
    rm $LOG_FILE
fi

kubectl logs "$pod_name" -f --all-containers=true |tee $LOG_FILE &
log_pid=$!

WRAP_APPLICATION_DONE=
WRAP_DATAMON_DONE=
while [[ -z $WRAP_APPLICATION_DONE || -z $WRAP_DATAMON_DONE ]]; do
    if cat $LOG_FILE | grep -q '^wrap_application sleeping indefinitely'; then
        WRAP_APPLICATION_DONE=true
    fi
    if cat $LOG_FILE | grep -q '^wrap_datamon_pg sleeping indefinitely'; then
        WRAP_DATAMON_DONE=true
    fi
    sleep 3
done

##### verify

## in-bundle paths
# dupe: wrap_datamon_pg.sh
BP_META=meta
BP_PG_VERSION=${BP_META}/pg_version
BP_DATA=data
BP_PG_TAR=${BP_DATA}/backup.tar.gz

##

MY_PG_PORT=5430
data_dir=/tmp/demo_pg_coord_verify_data_dir
mount_dir=/tmp/demo_pg_coord_verify_mount_dir
log_dir=/tmp/demo_pg_coord_verify_log_dir
blank_pg_dir=/tmp/demo_pg_coord_verify_data_dir_blank

if [[ -d ${data_dir} ]]; then
    rm -rf ${data_dir}
fi
mkdir -p ${data_dir}
if [[ -d ${mount_dir} ]]; then
    umount ${mount_dir} || true
    rm -rf ${mount_dir}
fi
mkdir -p ${mount_dir}
if [[ -d ${log_dir} ]]; then
    rm -rf ${log_dir}
fi
mkdir -p ${log_dir}
if [[ -d ${blank_pg_dir} ]]; then
    rm -rf ${blank_pg_dir}
fi
mkdir -p ${blank_pg_dir}

# --repo --label params from YAML
mount_params=(bundle mount \
                     --stream \
                     --mount $mount_dir \
                     --repo ransom-datamon-test-repo \
             )
if $init_demo_bundle; then
    # dupe: label from create_coord_pg_pod.sh
    mount_params=($mount_params --label pg-coord-example-input)
else
    mount_params=($mount_params --label pg-coord-example)
fi

"$DATAMON_EXEC" $mount_params > ${log_dir}/datamon_mount.log 2>&1 &
datamon_pid=$!
print -- 'datamon mount setup'
sleep 10
bundle_pg_version=$(cat ${mount_dir}/${BP_PG_VERSION})
# todo: verify version string in more detail
print -- "in-bundle pg version ${bundle_pg_version}"
(cd $data_dir && \
     >${log_dir}/untar.log 2>${log_dir}/untar_err.log \
     tar -xvf ${mount_dir}/${BP_PG_TAR})


# print "unpacked db from ${mount_dir}/${BP_PG_TAR} to $data_dir ," \
#       "return to con't"
# read

# todo: slay() in wrap_datamon_pg.sh remains unimpl
print -- 'datamon mount teardown'
kill $datamon_pid
sleep 10

# todo: workaround version mismatch to wrap_datamon_pg
PG_VERSION=$(postgres --version)
if [[ ! ${bundle_pg_version} = ${PG_VERSION} ]]; then
    print 'in-bundle pg version and desktop version mismatch'
    if [[ -f ${data_dir}/postgresql.conf ]]; then
        print -- 'workaround: use local pg configuration'
        initdb --no-locale -D ${blank_pg_dir}
        rm ${data_dir}/postgresql.conf
        cp ${blank_pg_dir}/postgresql.conf ${data_dir}/postgresql.conf
    fi
fi
# todo: perms bits to wrap_datamon_pg
chmod -R 750 ${data_dir}
postgres -D ${data_dir} -p ${MY_PG_PORT} > ${log_dir}/pg.log 2>&1 &
pg_pid=$!

print -- "blocking on postgres on ${MY_PG_PORT} " \
      "with pid ${pg_pid}" \
      ".. see ${log_dir}/pg.log"

while ! &>/dev/null psql \
        -h localhost \
        -p ${MY_PG_PORT} \
        -U postgres -l; do
    print "waiting on db start..."
    sleep 1
done

# sleep 10

# todo: psql-based verification

# echo "sleeping indefinitely (for debug)"
# while true; do sleep 100; done

get_tabla_idx_vals() {
    print 'select an_idx from tabla_e;' | \
        psql -p 5430 -U testpguser testdb | \
        awk '
BEGIN { on_row = 0 }
$0 ~ /^\(/ {if(on_row) {on_row = 0}}
{if(on_row) {print $1;}}
$0 ~ /^----/ { on_row = 1 }
'
}

tabla_idx_vals_scalar=$(get_tabla_idx_vals)
tabla_idx_vals_array=(${(f)tabla_idx_vals_scalar})

typeset -i expected_value
if $init_demo_bundle; then
    expected_value=5
else
    expected_value=$((5 + $SOME_CONST))
fi

if [[ ${tabla_idx_vals_array[3]} -eq ${expected_value} ]]; then
    print "got an expected value out of database"
else
    print "got unexpected value out of database artifact"
    print "${tabla_idx_vals_array[3]} != $((5 + $SOME_CONST))"
    exit 1
fi

# todo: slay
kill $pg_pid
