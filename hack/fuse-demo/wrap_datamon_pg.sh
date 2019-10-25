#! /bin/zsh

setopt ERR_EXIT
setopt PIPE_FAIL

#####

### util

terminate() {
    print -- "$*" 1>&2
    exit 1
}

dbg_print() {
    typeset dbg=true
    if $dbg; then
        print -- $*
    fi
}

#####

POLL_INTERVAL=1 # sec

PG_VERSION=$(postgres --version | \
                 sed 's/^.*(PostgreSQL) \([0-9]*\.[0-9]*\).*$/\1/')

PG_SU=postgres

STAGE_BASE=/pg_stage
PG_DATA_DIR_ROOT=${STAGE_BASE}/pg_data_dir
MNT_DIR_ROOT=${STAGE_BASE}/mounts
LOG_ROOT=${STAGE_BASE}/logs
UPLOAD_STAGE=${STAGE_BASE}/upload

# todo: populate from pod info via downward api
# https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/
CFG_EMAIL="pg_wrap@oneconcern.com"
CFG_NAME="pg_wrap"

## in-bundle paths
BP_META=meta
BP_PG_VERSION=${BP_META}/pg_version
BP_DATA=data
BP_PG_TAR=${BP_DATA}/backup.tar.gz

IGNORE_VERSION_MISMATCH=false

#####

typeset COORD_POINT
typeset -a PG_DB_IDS
typeset -A PG_DB_PORTS
# dest
typeset -A PG_DB_REPOS
typeset -A PG_DB_MSGS
typeset -A PG_DB_LABELS
# src
typeset -A PG_DB_HAS_SRC
typeset -A PG_DB_SRC_REPOS
typeset -A PG_DB_SRC_BUNDLES
typeset -A PG_DB_SRC_LABELS
# temp
typeset -i pg_db_idx
typeset pg_db_id

SLEEP_INSTEAD_OF_EXIT=

##

if [[ -n $dm_pg_params ]]; then
    typeset dm_pg_params_val
    env_vars_file=/tmp/env_vars_from_params.sh
    if [[ -f $dm_pg_params ]]; then
        dbg_print "parsing parameters YAML file $dm_pg_params"
        dm_pg_params_val=$(cat $dm_pg_params)
    else
        dbg_print "parsing parameters YAML from env var"
        dm_pg_params_val=$dm_pg_params
    fi
    print -- $dm_pg_params_val | \
        datamon_sidecar_param parse postgres > $env_vars_file
    . $env_vars_file
fi

##

deserialize_dict() {
    local item_sep
    local kv_sep
    local input_val
    typeset -A output_dict
    #
    local input_str
    local dict_name
    input_str=$1
    dict_name=$2
    #
    item_sep=$(print -- $input_str |sed 's/^\(.\).*$/\1/')
    kv_sep=$(print -- $input_str |sed 's/^.\(.\).*$/\1/')
    if [[ $item_sep = '.' ]]; then
        terminate "'.' is not a valid parameter seperator"
    fi
    if [[ $kv_sep = '.' ]]; then
        terminate "'.' is not a valid parameter seperator"
    fi
    input_val=$(print -- $input_str |sed 's/^..\(.*\)$/\1/')
    items=(${(ps.$item_sep.)input_val})
    for item in $items; do
        opt=$(print -- $item |cut -d $kv_sep -f 1)
        if print -- $item |grep -q $kv_sep; then
            arg=$(print -- $item |cut -d $kv_sep -f 2)
        else
            arg=true
        fi
        output_dict[$opt]=$arg
    done
    if [[ -z $dict_name ]]; then
        print -r -- ${(qkv)output_dict}
    else
        eval "${dict_name}=($(print -r -- ${(qkv)output_dict}))"
    fi
}

# parse labels into internal data structures,
# encapsulating kubernetes upward api details
typeset dm_pg_opts_global
typeset -A dm_pg_opts_dbs

env_vars_str=$(export)
env_vars_lines=(${(f)env_vars_str})
typeset -A env_vars
for env_var_line in $env_vars_lines; do
    env_var_name=$(print -- $env_var_line |cut -d '=' -f 1)
    env_var_contents=$(print -- $env_var_line |cut -d '=' -f 2)
    env_vars[$env_var_name]=${(Q)env_var_contents}
done

for env_var_name in ${(k)env_vars}; do
    if [[ $env_var_name = 'dm_pg_opts' ]]; then
        if [[ -n $dm_pg_opts_global ]]; then
            terminate "got duplicate global opts env_var"
        fi
        dm_pg_opts_global=$env_vars[$env_var_name]
        continue
    fi
    if print -- $env_var_name |grep -q '^dm_pg_db_'; then
        pg_db_id=$(print -- $env_var_name |sed 's/^dm_pg_db_//')
        dm_pg_opts_dbs[$pg_db_id]=$env_vars[$env_var_name]
    fi
done

# parse global opts data structure

typeset -A opts_global_dict
deserialize_dict $dm_pg_opts_global opts_global_dict

for opt in ${(k)opts_global_dict}; do
    arg=$opts_global_dict[$opt]
    case $opt in
        (S)
            SLEEP_INSTEAD_OF_EXIT=true
            ;;
        (V)
            IGNORE_VERSION_MISMATCH=$arg
            ;;
        (c)
            COORD_POINT=$arg
            ;;
        (\?)
            terminate "unknown global option '$opt'"
            ;;
    esac
done

for pg_db_id in ${(k)dm_pg_opts_dbs}; do
    PG_DB_IDS=($PG_DB_IDS $pg_db_id)
    typeset -A db_opts_dict
    deserialize_dict ${dm_pg_opts_dbs[$pg_db_id]} db_opts_dict
    PG_DB_PORTS[$pg_db_id]=$db_opts_dict[p]
    PG_DB_MSGS[$pg_db_id]=$db_opts_dict[m]
    PG_DB_LABELS[$pg_db_id]=$db_opts_dict[l]
    PG_DB_REPOS[$pg_db_id]=$db_opts_dict[r]
    PG_DB_SRC_LABELS[$pg_db_id]=$db_opts_dict[sl]
    PG_DB_SRC_REPOS[$pg_db_id]=$db_opts_dict[sr]
    PG_DB_SRC_BUNDLES[$pg_db_id]=$db_opts_dict[sb]
    if [[ -n $PG_DB_SRC_LABELS[$pg_db_id] || \
              -n $PG_DB_SRC_REPOS[$pg_db_id] || \
              -n $PG_DB_SRC_BUNDLES[$pg_db_id] ]]; then
        PG_DB_HAS_SRC[$pg_db_id]=true
    fi
done

##

for pg_db_id in $PG_DB_IDS; do
    if [[ -z ${COORD_POINT} ]]; then
        terminate 'coordination point unset'
    fi
    if [[ -z ${PG_DB_PORTS[$pg_db_id]} ]]; then
        terminate "missing port for $pg_db_id"
    fi
    if [[ -z ${PG_DB_REPOS[$pg_db_id]} ]]; then
        terminate "missing repo for $pg_db_id"
    fi
    if [[ -z ${PG_DB_MSGS[$pg_db_id]} ]]; then
        terminate "missing message for $pg_db_id"
    fi
    if [[ -n ${PG_DB_HAS_SRC[$pg_db_id]} ]]; then
        if [[ -z ${PG_DB_SRC_REPOS[$pg_db_id]} ]]; then
            terminate "missing source repo for $pg_db_id"
        fi
        if [[ -z ${PG_DB_SRC_BUNDLES[$pg_db_id]} && -z ${PG_DB_SRC_LABELS[$pg_db_id]} ]]; then
            terminate "no source data specified for $pg_db_id"
        fi
        if [[ -n ${PG_DB_SRC_BUNDLES[$pg_db_id]} && -n ${PG_DB_SRC_LABELS[$pg_db_id]} ]]; then
            terminate "specifying source data by bundleid or and labelid is mutually exclusive"
        fi
    fi
done

#####

### util

# #% =EVENT_NAME= <- wrap_application.sh
await_event() {
    COORD_DONE=
    EVENT_NAME="$1"
    DBG_MSG="$2"
    DBG_POLLS="$3"
    if [[ -n $DBG_MSG ]]; then
        echo "$DBG_MSG"
    fi
    while [[ -z $COORD_DONE ]]; do
        if [[ -f "${COORD_POINT}/${EVENT_NAME}" ]]; then
            COORD_DONE=1
        fi
        if [[ -n $DBG_POLLS ]]; then
            echo "... $DBG_MSG ..."
        fi
        sleep "$POLL_INTERVAL"
    done
}

#% wrap_application.sh <- =EVENT_NAME=
emit_event() {
    EVENT_NAME="$1"
    DBG_MSG="$2"
    echo "$DBG_MSG"
    touch "${COORD_POINT}/${EVENT_NAME}"
}

slay() {
    typeset -i pid
    typeset pids_str
    typeset -a pids_arr
    typeset -i num_tries
    typeset sent_term
    sent_term=false
    pid="$1"
    kill $pid
    while true; do
        if [[ num_tries -eq 10 ]]; then
           dbg_print "sending SIGTERM to $pid"
           kill -9 $pid
        fi
        pids_str=$(ps | awk 'NR > 1 { print $1 }')
        pids_arr=(${(f)pids_str})
        if ! ((${pids_arr[(Ie)$pid]})); then
            break
        fi
        dbg_print "awaiting $pid exit after signal"
        sleep 1
        ((num_tries++)) || true
    done
}

# placeholder for postgres-specific error-handling
stop_postgres() {
    pid="$1"
    slay $pid
}

#####

mkdir -p $PG_DATA_DIR_ROOT
mkdir -p $MNT_DIR_ROOT
mkdir -p $LOG_ROOT

##

datamon config create \
        --name $CFG_NAME \
        --email $CFG_EMAIL

##

dbg_print "setting privs on fuse device"
sudo chgrp developers /dev/fuse

dbg_print "starting postgres database processes"

typeset -a version_mismatched_pg_db_ids
typeset -A pg_pids
for pg_db_id in $PG_DB_IDS; do
    data_dir=${PG_DATA_DIR_ROOT}/${pg_db_id}
    if [[ -e ${data_dir} ]]; then
        terminate "data directory path $data_dir already exists"
    fi
    mkdir ${data_dir}
    # ??? could download the entire bundle upfront instead of mount?
    if [[ -n ${PG_DB_HAS_SRC[$pg_db_id]} ]]; then
        mount_dir=${MNT_DIR_ROOT}/${pg_db_id}
        mount_params=(bundle mount \
                             --stream \
                             --repo ${PG_DB_SRC_REPOS[$pg_db_id]} \
                             --mount $mount_dir)
        log_file_mount=${LOG_ROOT}/datamon_mount.${pg_db_id}.log
        if [[ -z ${PG_DB_SRC_BUNDLES[$pg_db_id]} ]]; then
            mount_params=($mount_params --label ${PG_DB_SRC_LABELS[$pg_db_id]})
        else
            mount_params=($mount_params --bundle ${PG_DB_SRC_BUNDLES[$pg_db_id]})
        fi
        if [[ -e $mount_dir ]]; then
            terminate "mount directory path $mount_dir already exists"
        fi
        mkdir $mount_dir
        unsetopt ERR_EXIT
        datamon $mount_params > $log_file_mount 2>&1 &
        datamon_status=$?
        datamon_pid=$!
        setopt ERR_EXIT
        if [[ ! $datamon_status -eq 0 ]]; then
            cat $log_file_mount
            terminate "error starting 'datamon $mount_params'"
        fi
        dbg_print "started 'datamon $mount_params' with log ${log_file_mount}"
        # block until mount found by os
        mount_waiting=true
        while $mount_waiting; do
            dbg_print "waiting on mount at $mount_dir"
            mount_data=$(mount | cut -d" " -f 3,5)
            if print "$mount_data" | grep -q "^$mount_dir fuse$"; then
                mount_waiting=false
                dbg_print "found mount at $mount_dir"
            fi
            sleep 1
        done
        # verify metadata
        version_path=${mount_dir}/${BP_PG_VERSION}
        if [[ -f ${version_path} ]]; then
            bundle_pg_version=$(cat ${version_path})
        else
            terminate "didn't find version file at ${version_path}"
        fi
        if [[ $bundle_pg_version != $PG_VERSION ]]; then
            if $IGNORE_VERSION_MISMATCH; then
                dbg_print "pg version mistmatch $bundle_pg_version -- $PG_VERSION"
                # todo: revisit what to do in case of postgres version bumps
                dbg_print "starting blank database instead"
                version_mismatched_pg_db_ids=($version_mismatched_pg_db_ids \
                                                  $pg_db_id)
                initdb --no-locale -D $data_dir
            else
                terminate "pg version mistmatch $bundle_pg_version -- $PG_VERSION"
            fi
        else
            (cd $data_dir && \
                 >${LOG_ROOT}/untar.${pg_db_id}.log \
                  2>${LOG_ROOT}/untar_err.${pg_db_id}.log \
                  tar -xvf ${mount_dir}/${BP_PG_TAR})
            slay $datamon_pid
            chmod -R 750 $data_dir
        fi
    else
        # --no-locale flag helps artifact portability
        # ??? other parms to set here?
        initdb --no-locale -D $data_dir
    fi
    log_file_pg=${LOG_ROOT}/pg.${pg_db_id}.log
    log_file_pg_err=${LOG_ROOT}/pg_err.${pg_db_id}.log
    unsetopt ERR_EXIT
    >${log_file_pg} 2>${log_file_pg_err} \
     postgres -D $data_dir -p $PG_DB_PORTS[$pg_db_id] &
    pg_pid=$!
    pg_status=$?
    setopt ERR_EXIT
    if [[ ! $pg_status -eq 0 ]]; then
        cat ${log_file_pg}
        cat ${log_file_pg_err}
        terminate "error starting postgres"
    fi
    pg_pids[$pg_db_id]=$pg_pid
    dbg_print "begin block on db id ${pg_db_id} start"
    if [[ -n ${PG_DB_HAS_SRC[$pg_db_id]} && \
              ${version_mismatched_pg_db_ids[(Ie)$pg_db_id]} -eq 0 ]]; then
        dbg_print "block on ${pg_db_id} db start by query"
        while ! &>/dev/null psql \
                -h localhost \
                -p $PG_DB_PORTS[$pg_db_id] \
                -U $PG_SU -l; do
            dbg_print "waiting on ${pg_db_id} db start (query)..."
            sleep $POLL_INTERVAL
        done
    else
        dbg_print "block on ${pg_db_id} db start by createuser"
        while ! &>/dev/null createuser -p $PG_DB_PORTS[$pg_db_id] -s $PG_SU; do
            dbg_print "waiting on ${pg_db_id} db start (createuser)..."
            sleep $POLL_INTERVAL
        done
    fi
done

dbg_print "postgres database processes started"

emit_event \
    'dbstarted' \
    'dispatching db started event'

await_event \
    'initdbupload' \
    'waiting on db upload event'

# VACUUM is a Postgres-specific SQL addition
# that allows the filesystem representation of the database
# to take less disk usage.
typeset -A db_name_to_username
for pg_db_id in $PG_DB_IDS; do
    db_name_to_username=()
    du_str=$(psql -p $PG_DB_PORTS[$pg_db_id] -U postgres -l | \
                 tail +4 | grep -v '^(.* rows)$' | \
                 grep -v '^ *$' | awk -F '|' '{print $1 $2}')
    for du_pair in ${(f)du_str}; do
        db_name=$(print -- $du_pair | awk '{print $1}')
        username=$(print -- $du_pair | awk '{print $2}')
        db_name_to_username[$db_name]=$username
    done
    for db_name in ${(k)db_name_to_username}; do
        if [[ $db_name == 'postgres' || \
                  $db_name == 'template0' || \
                  $db_name == 'template1' ]]; then
            continue
        fi
        print -- 'VACUUM;' | \
            psql \
                -h localhost \
                -p $PG_DB_PORTS[$pg_db_id] \
                -U ${db_name_to_username[$db_name]} \
                $db_name
        print -- 'CHECKPOINT;' | \
            psql \
                -h localhost \
                -p $PG_DB_PORTS[$pg_db_id] \
                -U $PG_SU \
                $db_name
    done
done

dbg_print "stopping postgres processes"

for pg_db_id in $PG_DB_IDS; do
    stop_postgres $pg_pids[$pg_db_id]
done

dbg_print "uploading data directories"

if [[ -e ${UPLOAD_STAGE} ]]; then
    terminate "upload staging area ${UPLOAD_STAGE} already exists"
fi

for pg_db_id in $PG_DB_IDS; do
    mkdir -p ${UPLOAD_STAGE}
    mkdir ${UPLOAD_STAGE}/${BP_META}
    mkdir ${UPLOAD_STAGE}/${BP_DATA}
    dbg_print "prepare staging area"
    data_dir=${PG_DATA_DIR_ROOT}/${pg_db_id}
    (cd $data_dir && \
       >${LOG_ROOT}/tar.${pg_db_id}.log 2>${LOG_ROOT}/tar_err.${pg_db_id}.log \
         tar -cvf ${UPLOAD_STAGE}/${BP_PG_TAR} *)
    print -- ${PG_VERSION} > ${UPLOAD_STAGE}/${BP_PG_VERSION}
    log_file_upload=${LOG_ROOT}/datamon_upload.${upload_idx}.log
    upload_params=(bundle upload \
                          --path ${UPLOAD_STAGE} \
                          --message $PG_DB_MSGS[$pg_db_id] \
                          --repo $PG_DB_REPOS[$pg_db_id])
    if [[ -n $PG_DB_LABELS[$pg_db_id] ]]; then
        upload_params=($upload_params --label $PG_DB_LABELS[$pg_db_id])
    fi
    dbg_print "perform upload 'datamon $upload_params' with log file at '$log_file_upload'"
    unsetopt ERR_EXIT
    datamon $upload_params > $log_file_upload 2>&1
    datamon_status=$?
    setopt ERR_EXIT
    if [[ ! $datamon_status -eq 0 ]]; then
        cat $log_file_upload
        terminate "upload command failed"
    fi
    dbg_print "upload command had nominal status"
    rm -rf ${UPLOAD_STAGE}
done

emit_event \
    'dbuploaddone' \
    'dispatching db upload done event'

if [[ -z $SLEEP_INSTEAD_OF_EXIT ]]; then
    exit 0
fi

echo "wrap_datamon_pg sleeping indefinitely (for debug)"
while true; do sleep 100; done
