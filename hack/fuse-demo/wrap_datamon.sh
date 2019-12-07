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

### datamon wrapper (demo)
# half of the container coordination sketch, a script like this one
# is meant to wrap datamon in the sidecar container of an Argo DAG node
# and communciate with a script like wrap_application.sh.

POLL_INTERVAL=1 # sec

#####

typeset COORD_POINT

typeset CONTEXT_BUCKET_NAME
typeset CONTEXT_NAME

SLEEP_INSTEAD_OF_EXIT=

####
# bridge parameters to shell script specific format
# (currently environment variables

if [[ -n $dm_fuse_params ]]; then
    typeset dm_fuse_params_val
    env_vars_file=/tmp/env_vars_from_params.sh
    if [[ -f $dm_fuse_params ]]; then
        dbg_print "parsing parameters YAML file $dm_fuse_params"
        dm_fuse_params_val=$(cat $dm_fuse_params)
    else
        dbg_print "parsing parameters YAML from env var"
        dm_fuse_params_val=$dm_fuse_params
    fi
    print -- $dm_fuse_params_val | \
        datamon_sidecar_param parse fuse > $env_vars_file
    . $env_vars_file
fi

####

# parse the shell script specific format

# deserialize an associate array from a scalar
# example usage
# typeset -A opts_global_dict
# opts_global_dict=($(deserialize_dict $opts_global))
# todo: implement via set variables by reference as in
# deserialize_dict $opts_global opts_global_dict
#   .. access by reference is via ${(P)var_name}.
#   unsure how to set.
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

typeset dm_fuse_opts_global
typeset -A dm_fuse_opts_bds

env_vars_str=$(export)
env_vars_lines=(${(f)env_vars_str})
typeset -A env_vars
for env_var_line in $env_vars_lines; do
    env_var_name=$(print -- $env_var_line |cut -d '=' -f 1)
    env_var_contents=$(print -- $env_var_line |cut -d '=' -f 2)
    env_vars[$env_var_name]=${(Q)env_var_contents}
done

for env_var_name in ${(k)env_vars}; do
    if [[ $env_var_name = 'dm_fuse_opts' ]]; then
        if [[ -n $dm_fuse_opts_global ]]; then
            terminate "got duplicate global opts env_var"
        fi
        dm_fuse_opts_global=$env_vars[$env_var_name]
        continue
    fi
    if print -- $env_var_name |grep -q '^dm_fuse_bd_'; then
        dm_v_id=$(print -- $env_var_name |sed 's/^dm_fuse_bd_//')
        dm_fuse_opts_bds[$dm_v_id]=$env_vars[$env_var_name]
    fi
done

##

# populate internal data structures from environment variables

typeset -A opts_global_dict
deserialize_dict $dm_fuse_opts_global opts_global_dict

if [[ -n $opts_global_dict[c] ]]; then
    COORD_POINT=$opts_global_dict[c]
fi
if [[ -n $opts_global_dict[S] ]]; then
    SLEEP_INSTEAD_OF_EXIT=true
fi

CONTEXT_BUCKET_NAME=$opts_global_dict[b]
if [[ -z $CONTEXT_BUCKET_NAME ]]; then
    terminate "config bucket not set"
fi
CONTEXT_NAME=$opts_global_dict[a]
if [[ -z $CONTEXT_NAME ]]; then
    terminate "context not set"
fi

typeset -a SIDECAR_VERTEX_IDS
# dest
typeset -A DATAMON_DEST_PATHS
typeset -A DATAMON_DEST_REPOS
typeset -A DATAMON_DEST_MSGS
typeset -A DATAMON_DEST_LABELS
typeset -A DATAMON_DEST_BUNDLE_ID_FILES
# src
typeset -A DATAMON_SRC_PATHS
typeset -A DATAMON_SRC_REPOS
typeset -A DATAMON_SRC_BUNDLES
typeset -A DATAMON_SRC_LABELS

typeset -A bd_opts_dict
for dm_v_id in ${(k)dm_fuse_opts_bds}; do
    SIDECAR_VERTEX_IDS=($SIDECAR_VERTEX_IDS $dm_v_id)
    deserialize_dict ${dm_fuse_opts_bds[$dm_v_id]} bd_opts_dict
    DATAMON_DEST_PATHS[$dm_v_id]=$bd_opts_dict[dp]
    DATAMON_DEST_REPOS[$dm_v_id]=$bd_opts_dict[dr]
    DATAMON_DEST_MSGS[$dm_v_id]=$bd_opts_dict[dm]
    DATAMON_DEST_LABELS[$dm_v_id]=$bd_opts_dict[dl]
    DATAMON_DEST_BUNDLE_ID_FILES[$dm_v_id]=$bd_opts_dict[dif]
    DATAMON_SRC_PATHS[$dm_v_id]=$bd_opts_dict[sp]
    DATAMON_SRC_REPOS[$dm_v_id]=$bd_opts_dict[sr]
    DATAMON_SRC_BUNDLES[$dm_v_id]=$bd_opts_dict[sb]
    DATAMON_SRC_LABELS[$dm_v_id]=$bd_opts_dict[sl]
done

# verify internal data structures
for dm_v_id in $SIDECAR_VERTEX_IDS; do
    if [[ -z ${DATAMON_SRC_PATHS[$dm_v_id]} && \
              -z ${DATAMON_DEST_PATHS[$dm_v_id]} ]]; then
        terminate "neither source nor destination specified for ${dm_v_id}"
    fi
    if [[ -n ${DATAMON_SRC_PATHS[$dm_v_id]} ]]; then
        if [[ -n ${DATAMON_DEST_PATHS[$dm_v_id]} ]]; then
            terminate "specified both source and destination path for ${dm_v_id}"
        fi
        if [[ -n ${DATAMON_DEST_REPOS[$dm_v_id]} ]]; then
            terminate "destination repo present on source vertex ${dm_v_id}"
        fi
        if [[ -n ${DATAMON_DEST_MSGS[$dm_v_id]} ]]; then
            terminate "destination message present on source vertex ${dm_v_id}"
        fi
        if [[ -n ${DATAMON_DEST_LABELS[$dm_v_id]} ]]; then
            terminate "destination label present on source vertex ${dm_v_id}"
        fi
    fi
    if [[ -n ${DATAMON_DEST_PATHS[$dm_v_id]} ]]; then
        if [[ -n ${DATAMON_SRC_PATHS[$dm_v_id]} ]]; then
            terminate "specified both source and destination path for ${dm_v_id}"
        fi
        if [[ -n ${DATAMON_SRC_REPOS[$dm_v_id]} ]]; then
            terminate "source repo present on destination vertex ${dm_v_id}"
        fi
        if [[ -n ${DATAMON_SRC_MSGS[$dm_v_id]} ]]; then
            terminate "source message present on destination vertex ${dm_v_id}"
        fi
        if [[ -n ${DATAMON_SRC_LABELS[$dm_v_id]} ]]; then
            terminate "source label present on destination vertex ${dm_v_id}"
        fi
    fi
    if [[ -n ${DATAMON_SRC_PATHS[$dm_v_id]} ]]; then
        if [[ -z ${DATAMON_SRC_REPOS[$dm_v_id]} ]]; then
            terminate "missing repo for $dm_v_id"
        fi
        if [[ -z ${DATAMON_SRC_BUNDLES[$dm_v_id]} && \
                  -z ${DATAMON_SRC_LABELS[$dm_v_id]} ]]; then
            terminate "no source data specified for $dm_v_id"
        fi
        if [[ -n ${DATAMON_SRC_BUNDLES[$dm_v_id]} && \
                  -n ${DATAMON_SRC_LABELS[$dm_v_id]} ]]; then
            terminate "specifying source data by bundleid or and labelid is mutually exclusive"
        fi
    fi
    if [[ -n ${DATAMON_DEST_PATHS[$dm_v_id]} ]]; then
        if [[ -z ${DATAMON_DEST_REPOS[$dm_v_id]} ]]; then
            terminate "missing repo for $dm_v_id"
        fi
        if [[ -z ${DATAMON_DEST_MSGS[$dm_v_id]} ]]; then
            terminate "missing message for $dm_v_id"
        fi
    fi
done

if [[ $#SIDECAR_VERTEX_IDS -eq 0 ]]; then
    terminate "no sources or destinations specified"
fi
if [[ -z $COORD_POINT ]]; then
    terminate "coordination point not set"
fi

print -- 'internal data structures verified'

for dm_v_id in $SIDECAR_VERTEX_IDS; do
    BUNDLE_ID_FILE=$DATAMON_DEST_BUNDLE_ID_FILES[$dm_v_id]
    if [[ -n "$BUNDLE_ID_FILE" ]]; then
        bundle_id_file_dir=$(dirname "$BUNDLE_ID_FILE")
        if [[ ! -d "$bundle_id_file_dir" ]]; then
            mkdir -p "$bundle_id_file_dir"
        fi
        if [[ -f "$BUNDLE_ID_FILE" ]]; then
            terminate "$BUNDLE_ID_FILE already exists"
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

dbg_print "have zsh version $ZSH_VERSION"
# todo: review use of this env variable in golang binary
export DATAMON_GLOBAL_CONFIG=$CONTEXT_BUCKET_NAME
dbg_print "set DATAMON_GLOBAL_CONFIG to '${DATAMON_GLOBAL_CONFIG}' for datamon binary"

run_datamon_cmd() {
    typeset params_param
    typeset -a params
    typeset stat
    params_param="$1"
    # split string according to shell parsing, remove quotes
    # http://zsh.sourceforge.net/Doc/Release/Expansion.html#Parameter-Expansion-Flags
    params=(${(Q)${(z)params_param}})
    print -- 'inbound run_datamon_cmd datamon binary params'
    print -- '===='
    print -l -- $params
    print -- '===='

    params=($params --context $CONTEXT_NAME)
    print -- 'running datamon with params'
    print -- '===='
    print -l -- $params
    print -- '===='

    datamon $params
    return $?
}

unsetopt ERR_EXIT
run_datamon_cmd 'context get'
datamon_status=$?
setopt ERR_EXIT
if [[ $datamon_status -eq 2 ]]; then
    1>&2 print -- "didn't find $CONTEXT_NAME in bucket $CONTEXT_BUCKET_NAME"
    exit 1
fi
if [[ ! $datamon_status -eq 0 ]]; then
    1>&2 print -- "context get failed for unknown reason"
fi

run_datamon_cmd 'config create'

## COORDINATION BEGINS by starting a datamon FUSE mount
print -- "starting sources"

# in some kubernetes distros like docker-desktop, /dev/fuse has perms 660 rather than 666
dbg_print "setting privs on fuse device"
if [[ -e /dev/fuse ]]; then
    sudo chgrp developers /dev/fuse
else
    1>&2 print -- "didn't find fuse device, /dev/fuse"
    exit 1
fi

typeset -a mount_points
typeset -i mount_idxp
typeset -a datamon_pids

for dm_v_id in $SIDECAR_VERTEX_IDS; do
    mount_point=${DATAMON_SRC_PATHS[$dm_v_id]}
    if [[ -z ${mount_point} ]]; then
        continue
    fi
    if [[ ! -d ${mount_point} ]]; then
        terminate "'$mount_point' doesn't exist"
    fi

    mount_cmd_params=()
    mount_cmd_params=(bundle mount --stream \
                               --mount ${mount_point} \
                               --repo ${DATAMON_SRC_REPOS[$dm_v_id]} \
                       )
    if [[ -n ${DATAMON_SRC_LABELS[$dm_v_id]} ]]; then
        mount_cmd_params=($mount_cmd_params \
                                --label ${DATAMON_SRC_LABELS[$dm_v_id]})
    else
        mount_cmd_params=($mount_cmd_params \
                                --bundle ${DATAMON_SRC_BUNDLES[$dm_v_id]})
    fi

    dbg_print "running mount command '${mount_cmd_params}' (${mount_idx})"
    log_file_mount="/tmp/datamon_mount.${mount_idx}.log"
    unsetopt ERR_EXIT
    run_datamon_cmd "$mount_cmd_params" 2>&1 > >(tee -a "$log_file_mount") 2>&1 &
    datamon_status=$?
    datamon_pid=$!
    setopt ERR_EXIT
    echo "started datamon '${mount_cmd_params}' with pid ${datamon_pid}"

    if [[ ! $datamon_status -eq 0 ]]; then
        print -- "error starting datamon ${mount_cmd}, try shell" 2>&1
        cat "$log_file_mount"
        sleep 3600
        exit 1
    fi
    dbg_print "datamon status checks out okay"
    ((mount_idx++)) || true
    dbg_print "mount idx incremented"
    datamon_pids=($datamon_pid $datamon_pids)
    dbg_print "updated datamon pids"
    mount_points=($mount_point $mount_points)
    dbg_print "updated datamon mount points"
done

dbg_print "out of mount for loop"

dbg_print "started datamon mounts with pids '${datamon_pids},' mount_points '${mount_points}'"

dbg_print "waiting on datamon mount (datamon wrap)"

typeset -a found_mount_points
MOUNT_COORD_DONE=
while [[ -z $MOUNT_COORD_DONE ]]; do
    mount_data=$(mount | cut -d" " -f 3,5)
    for mount_point in $mount_points; do
        if ((${found_mount_points[(Ie)$mount_point]})); then
            # mount point already found
            continue
        fi
        if echo "$mount_data" | grep -q "^$mount_point fuse$"; then
            found_mount_points=($mount_point $found_mount_points)
        fi
    done
    dbg_print "${#found_mount_points} / ${#mount_points} fuse mounts found"
    if [[ ${#found_mount_points} -eq ${#mount_points} ]]; then
        MOUNT_COORD_DONE=1
    fi
    sleep "$POLL_INTERVAL"
done

dbg_print "datamon mount coordination done (datamon wrap)"

emit_event \
    'mountdone' \
    'dispatching mount done event'

await_event \
    'initupload' \
    'waiting on upload event'

## discard the FUSE mount, perform the upload

dbg_print "sending signal to stop datamon mount processes $datamon_pids"

for datamon_pid in $datamon_pids; do
    kill "$datamon_pid"
done

dbg_print "starting datamon upload"

## notify the the application if the upload was successful, and exit this container in any case

typeset -i upload_idx

for dm_v_id in $SIDECAR_VERTEX_IDS; do
    if [[ -z ${DATAMON_DEST_PATHS[$dm_v_id]} ]]; then
        continue
    fi
    upload_cmd_params=()
    upload_cmd_params=(bundle upload \
                               --path ${DATAMON_DEST_PATHS[$dm_v_id]} \
                               --message \
                               "\"${DATAMON_DEST_MSGS[$dm_v_id]}\"" \
                               --repo ${DATAMON_DEST_REPOS[$dm_v_id]} \
                       )
    if [[ -n ${DATAMON_DEST_LABELS[$dm_v_id]} ]]; then
        upload_cmd_params=($upload_cmd_params \
                                --label ${DATAMON_DEST_LABELS[$dm_v_id]})
    fi
    dbg_print "running upload command '${upload_cmd_params}' (${upload_idx})"
    log_file_upload="/tmp/datamon_upload.${upload_idx}.log"
    unsetopt ERR_EXIT
    run_datamon_cmd "$upload_cmd_params" > >(tee -a "$log_file_upload") 2>&1
    datamon_status=$?
    setopt ERR_EXIT
    if [[ ! $datamon_status -eq 0 ]]; then
        dbg_print "upload command failed"
        echo "error starting datamon $upload_cmd, try shell" 2>&1
        cat "$log_file_upload"
        sleep 3600
        exit 1
    fi
    dbg_print "upload command had nominal status"

    BUNDLE_ID_FILE=$DATAMON_DEST_BUNDLE_ID_FILES[$dm_v_id]
    if [[ -n $BUNDLE_ID_FILE ]]; then
        if [[ -f "$BUNDLE_ID_FILE" ]]; then
            terminate "$BUNDLE_ID_FILE already exists"
        fi
        # extracting bundle id from log (could be cleaner :/ error handling
        dbg_print "getting bundle id lines"
        unsetopt ERR_EXIT
        bundle_id_lines=$(cat "$log_file_upload" | grep 'Uploaded bundle id')
        bundle_id_lines_status=$?
        setopt ERR_EXIT
        if [[ ! $bundle_id_lines_status -eq 0 ]]; then
            print "didn't find any bundle id lines" 1>&2
            sleep 3600
            exit 1
        fi
        dbg_print "have bundle id lines ${bundle_id_lines}"
        unsetopt ERR_EXIT
        bundle_id=$(print -- ${bundle_id_lines} | \
                        tail -1 | \
                        sed 's/Uploaded bundle id:\(.*\)/\1/')
        bundle_id_status=$?
        setopt ERR_EXIT
        if [[ ! $bundle_id_status -eq 0 ]]; then
            print "didn't parse bundle id out of ${bundle_id_lines}" 1>&2
            sleep 3600
            exit 1
        fi
        echo "$bundle_id" > "$BUNDLE_ID_FILE"
    fi
    ((upload_idx++)) || true
done

emit_event \
    'uploaddone' \
    'dispatching upload done event'

if [[ -z $SLEEP_INSTEAD_OF_EXIT ]]; then
    exit 0
fi

dbg_print "wrap_datamon sleeping indefinitely (for debug)"
while true; do sleep 100; done
