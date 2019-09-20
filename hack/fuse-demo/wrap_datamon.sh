#! /bin/zsh

### datamon wrapper (demo)
# half of the container coordination sketch, a script like this one
# is meant to wrap datamon in the sidecar container of an Argo DAG node
# and communciate with a script like wrap_application.sh.

setopt ERR_EXIT
setopt PIPE_FAIL

POLL_INTERVAL=1 # sec

#####

typeset COORD_POINT
typeset -a DATAMON_CMDS
typeset BUNDLE_ID_FILE

SLEEP_INSTEAD_OF_EXIT=

while getopts sc:d:i: opt; do
    case $opt in
        (s)
            SLEEP_INSTEAD_OF_EXIT=true
            ;;
        (c)
            COORD_POINT="$OPTARG"
            ;;
        (d)
            datamon_cmd=$(echo "$OPTARG" |tr '\t' ' ' |tr -s ' ' | \
                              sed 's/^ //' |sed 's/ $//')
            DATAMON_CMDS=($datamon_cmd $DATAMON_CMDS)
            ;;
        (i)
            BUNDLE_ID_FILE="$OPTARG"
            ;;
        (\?)
            echo "Bad option, aborting."
            return 1
            ;;
    esac
done
if [[ "$OPTIND" -gt 1 ]]; then
    shift $(( OPTIND - 1 ))
fi

if [[ $#DATAMON_CMDS -eq 0 ]]; then
    echo "no datamon cmds to run" 1>&2
    exit 1
fi
if [[ -z $COORD_POINT ]]; then
    echo "coordination point not set" 1>&2
    exit 1
fi

echo "getopts checks ok"

typeset -a UPLOAD_CMDS
typeset -a MOUNT_CMDS
typeset CONFIG_CMD

for datamon_cmd in $DATAMON_CMDS; do
    if print $datamon_cmd | grep -q '^bundle'; then
        bundle_cmd_name=$(echo "$datamon_cmd" |sed 's/[^ ]* \([^ ]*\).*/\1/')
        case $bundle_cmd_name in
            (upload)
                UPLOAD_CMDS=($datamon_cmd $UPLOAD_CMDS)
                ;;
            (mount)
                MOUNT_CMDS=($datamon_cmd $MOUNT_CMDS)
                ;;
            (\?)
                echo "unsupported datamon bundle command '$bundle_cmd_name' " \
                     "in '$datamon_cmd'" 1>&2
                exit 1
                ;;
        esac
        continue
    fi
    if ! print $datamon_cmd | grep -q '^config'; then
        echo "only bundle and config commands supported.  got '$datamon_cmd'." 1>&2
        exit 1
    fi
    if [[ -n $CONFIG_CMD ]]; then
        echo "expected at most one 'config' command" 1>&2
        exit 1
    fi
    CONFIG_CMD="$datamon_cmd"
done

if [[ -n "$BUNDLE_ID_FILE" ]]; then
    bundle_id_file_dir=$(dirname "$BUNDLE_ID_FILE")
    if [[ ! -d "$bundle_id_file_dir" ]]; then
        mkdir -p "$bundle_id_file_dir"
    fi
    if [[ -f "$BUNDLE_ID_FILE" ]]; then
        echo "$BUNDLE_ID_FILE already exists" 1>&2
        exit 1
    fi
    if [[ ! $#UPLOAD_CMDS -eq 1 ]]; then
        echo "expected precisley one upload command when bundle id file specified" 1>&2
        exit 1
    fi
fi

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

run_datamon_cmd() {
    typeset params_param
    typeset -a params
    typeset stat
    params_param="$1"
    # split string according to shell parsing, remove quotes
    # http://zsh.sourceforge.net/Doc/Release/Expansion.html#Parameter-Expansion-Flags
    params=(${(Q)${(z)params_param}})
    datamon $params
    return $?
}

dbg_print() {
    typeset dbg=true
    if $dbg; then
        print -- $*
    fi
}

dbg_print "have zsh version $ZSH_VERSION"

if [[ -n $CONFIG_CMD ]]; then
    run_datamon_cmd "$CONFIG_CMD"
fi

## COORDINATION BEGINS by starting a datamon FUSE mount
echo "starting ${#MOUNT_CMDS} mounts '$MOUNT_CMDS'"

# in some kubernetes distros like docker-desktop, /dev/fuse has perms 660 rather than 666
echo "setting privs on fuse device"
sudo chgrp developers /dev/fuse

typeset -a mount_points
typeset -i mount_idx
typeset -a datamon_pids
for mount_cmd in $MOUNT_CMDS; do
    dbg_print "running mount command '${mount_cmd}' (${mount_idx})"
    mount_cmd_params=(${(Q)${(z)mount_cmd}})
    mount_flag_idx=${mount_cmd_params[(ie)--mount]}
    if [[ $mount_flag_idx -ge ${#mount_cmd_params} ]]; then
        print -- "didn't find --mount flag parameter in '$mount_cmd'" 1>&2
        exit 1
    fi
    mount_point=${mount_cmd_params[$((mount_flag_idx + 1))]}
    if [[ -z $mount_point ]]; then
        echo "didn't find mount point in '$mount_cmd'" 1>&2
        exit 1
    fi
    if [[ ! -d "$mount_point" ]]; then
        echo "'$mount_point' doesn't exist" 1>&2
        exit 1
    fi
    log_file_mount="/tmp/datamon_mount.${mount_idx}.log"
    unsetopt ERR_EXIT
    run_datamon_cmd $mount_cmd > "$log_file_mount" 2>&1 &
    datamon_status=$?
    datamon_pid=$!
    setopt ERR_EXIT
    echo "started datamon '$mount_cmd' with pid $datamon_pid"

    if [[ ! $datamon_status -eq 0 ]]; then
        echo "error starting datamon $mount_cmd, try shell" 2>&1
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

echo "started datamon mounts with pids '${datamon_pids},' mount_points '${mount_points}'"

echo "waiting on datamon mount (datamon wrap)"

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
    echo "${#found_mount_points} / ${#mount_points} fuse mounts found"
    if [[ ${#found_mount_points} -eq ${#mount_points} ]]; then
        MOUNT_COORD_DONE=1
    fi
    sleep "$POLL_INTERVAL"
done

echo "datamon mount coordination done (datamon wrap)"

emit_event \
    'mountdone' \
    'dispatching mount done event'

await_event \
    'initupload' \
    'waiting on upload event'

## discard the FUSE mount, perform the upload

echo "sending signal to stop datamon mount processes $datamon_pids"

for datamon_pid in $datamon_pids; do
    kill "$datamon_pid"
done

echo "starting datamon upload"

## notify the the application if the upload was successful, and exit this container in any case

typeset -i upload_idx
for upload_cmd in $UPLOAD_CMDS; do
    dbg_print "running upload command '${upload_cmd}' (${upload_idx})"
    log_file_upload="/tmp/datamon_upload.${upload_idx}.log"
    unsetopt ERR_EXIT
    run_datamon_cmd "$upload_cmd" > "$log_file_upload" 2>&1
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
    if [[ -n $BUNDLE_ID_FILE ]]; then
        if [[ -f "$BUNDLE_ID_FILE" ]]; then
            echo "$BUNDLE_ID_FILE already exists" 1>&2
            exit 1
        fi
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

echo "wrap_datamon sleeping indefinitely (for debug)"
while true; do sleep 100; done
