#! /bin/sh

### datamon wrapper (demo)
# half of the container coordination sketch, a script like this one
# is meant to wrap datamon in the sidecar container of an Argo DAG node
# and communciate with a script like wrap_application.sh.

POLL_INTERVAL=1 # sec

#####

COORD_POINT=
DATAMON_CMDS=
BUNDLE_ID_FILE=

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
            DATAMON_CMDS="${DATAMON_CMDS}:${datamon_cmd}"
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
if [ "$OPTIND" -gt 1 ]; then
    shift $(( OPTIND - 1 ))
fi

DATAMON_CMDS=$(echo "$DATAMON_CMDS" |sed 's/^://')

if [ -z "$DATAMON_CMDS" ]; then
    echo "no datamon cmds to run" 1>&2
    exit 1
fi
if [ -z "$COORD_POINT" ]; then
    echo "coordination point not set" 1>&2
    exit 1
fi

echo "getopts checks ok"

num_upload_cmds=0
UPLOAD_CMDS=
MOUNT_CMDS=
CONFIG_CMD=

ifs_orig="$IFS"
IFS=':'
for datamon_cmd in $DATAMON_CMDS; do
    if ! echo "$datamon_cmd" | grep -q '^bundle'; then
        if ! echo "$datamon_cmd" | grep -q '^config'; then
            echo "only bundle and config commands supported.  got '$datamon_cmd'." 1>&2
            exit 1
        else
            if [ -n "$CONFIG_CMD" ]; then
                echo "expected at most one 'config' command" 1>&2
                exit 1
            fi
            CONFIG_CMD="$datamon_cmd"
        fi
        else
            bundle_cmd_name=$(echo "$datamon_cmd" |sed 's/[^ ]* \([^ ]*\).*/\1/')
            case $bundle_cmd_name in
                (upload)
                    num_upload_cmds="$((num_upload_cmds + 1))"
                    UPLOAD_CMDS="$UPLOAD_CMDS:$datamon_cmd"
                    ;;
                (mount)
                    MOUNT_CMDS="$MOUNT_CMDS:$datamon_cmd"
                    ;;
                (\?)
                    echo "unsupported datamon bundle command '$bundle_cmd_name' " \
                         "in '$datamon_cmd'" 1>&2
                    exit 1
                    ;;
            esac
    fi
done
IFS="$ifs_orig"

if [ -n "$BUNDLE_ID_FILE" ]; then
    bundle_id_file_dir="$(dirname "$BUNDLE_ID_FILE")"
    if [ ! -d "$bundle_id_file_dir" ]; then
        mkdir -p "$bundle_id_file_dir"
    fi
    if [ -f "$BUNDLE_ID_FILE" ]; then
        echo "$BUNDLE_ID_FILE already exists" 1>&2
        exit 1
    fi
    if [ ! "$num_upload_cmds" -eq 1 ]; then
        echo "expected precisley one upload command when bundle id file specified" 1>&2
        exit 1
    fi
fi

UPLOAD_CMDS=$(echo "$UPLOAD_CMDS" |sed 's/^://')
MOUNT_CMDS=$(echo "$MOUNT_CMDS" |sed 's/^://')

#####

### util

# #% =EVENT_NAME= <- wrap_application.sh
await_event() {
    COORD_DONE=
    EVENT_NAME="$1"
    DBG_MSG="$2"
    DBG_POLLS="$3"
    if [ -n "$DBG_MSG" ]; then
        echo "$DBG_MSG"
    fi
    while [ -z "$COORD_DONE" ]; do
        if [ -f "${COORD_POINT}/${EVENT_NAME}" ]; then
            COORD_DONE=1
        fi
        if [ -n "$DBG_POLLS" ]; then
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
    params_param="$1"
    set --
    params_param_rem=$(echo "$params_param" |tr '\t' ' ' |tr -s ' ')
    while [ -n "$params_param_rem" ]; do
        params_param_rem=$(echo "$params_param_rem" |sed 's/^ //')
        if echo "$params_param_rem" |grep -q "^'"; then
            next_param=$(echo "$params_param_rem" |sed "s/^'\\([^']*\\)'.*/\\1/")
            if [ -z "$next_param" ]; then
                echo "unterminated quote in params string" 1>&2
                exit 1
            fi
            params_param_rem=$(echo "$params_param_rem" |sed "s/^'[^']*'//")
        else
            if echo "$params_param_rem" |grep -q '^"'; then
                next_param=$(echo "$params_param_rem" |sed 's/^"\([^"]*\)".*/\1/')
                if [ -z "$next_param" ]; then
                    echo "unterminated quote in params string" 1>&2
                    exit 1
                fi
                params_param_rem=$(echo "$params_param_rem" |sed 's/^"[^"]*"//')
            else
                next_param=$(echo "$params_param_rem" |sed 's/^\([^ ]*\).*/\1/')
                params_param_rem=$(echo "$params_param_rem" |sed 's/^[^ ]*//')
            fi
        fi
        set -- "$@" "$next_param"
    done
    datamon "$@"
    stat="$?"
    if [ "$stat" != '0' ]; then
        return "$stat"
    fi
}

if [ -n "$CONFIG_CMD" ]; then
    run_datamon_cmd "$CONFIG_CMD"
fi

## COORDINATION BEGINS by starting a datamon FUSE mount
echo "starting mounts '$MOUNT_CMDS'"

mount_points=
mount_idx=0
datamon_pids=
ifs_orig="$IFS"
IFS=':'
for mount_cmd in $MOUNT_CMDS; do
    mount_point=$(echo "$mount_cmd" |sed 's/^.*--mount//' |sed 's/^ *//' |sed 's/ .*//')
    if [ -z "$mount_point" ]; then
        echo "didn't find mount point in '$mount_cmd'" 1>&2
        exit
    fi
    if [ ! -d "$mount_point" ]; then
        echo "'$mount_point' doesn't exist" 1>&2
        exit
    fi
    IFS="$ifs_orig"
    log_file_mount="/tmp/datamon_mount.${mount_idx}.log"
    run_datamon_cmd "$mount_cmd" > "$log_file_mount" 2>&1 &
    IFS=':'
    datamon_status="$?"
    datamon_pid="$!"
    echo "started datamon '$mount_cmd' with pid $datamon_pid"

    if [ "$datamon_status" != "0" ]; then
        echo "error starting datamon $mount_cmd, try shell" 2>&1
        cat "$log_file_mount"
        sleep 3600
        exit 1
    fi
    mount_idx=$((mount_idx + 1))
    datamon_pids="$datamon_pids $datamon_pid"
    mount_points="$mount_points:$mount_point"
done
IFS="$ifs_orig"

datamon_pids=$(echo "$datamon_pids" |sed 's/^ //')
mount_points=$(echo "$mount_points" |sed 's/^://')

echo "started datamon mounts with pids '$datamon_pids,' mount_points '$mount_points'"

echo "waiting on datamon mount (datamon wrap)"

MOUNT_COORD_DONE=
while [ -z "$MOUNT_COORD_DONE" ]; do
    found_mount_idx=0
    mount_data=$(mount | cut -d" " -f 3,5)
    ifs_orig="$IFS"
    IFS=':'
    for mount_point in $mount_points; do
        if echo "$mount_data" | grep -q "^$mount_point fuse$"; then
            found_mount_idx=$((found_mount_idx + 1))
        fi
    done
    IFS="$ifs_orig"
    echo "$found_mount_idx / $mount_idx fuse mounts found"
    if [ "$mount_idx" = "$found_mount_idx" ]; then
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

upload_idx=0
ifs_orig="$IFS"
IFS=':'
for upload_cmd in $UPLOAD_CMDS; do
    IFS="$ifs_orig"
    log_file_upload="/tmp/datamon_upload.${upload_idx}.log"
    bundle_id=$(2>&1 run_datamon_cmd "$upload_cmd" | \
                    tee "$log_file_upload" | \
                    grep 'Uploaded bundle id' | \
                    tail -1 | \
                    sed 's/Uploaded bundle id:\(.*\)/\1/')
    datamon_status="$?"
    if [ "$datamon_status" != 0 ]; then
        echo "error starting datamon $upload_cmd, try shell" 2>&1
        cat "$log_file_upload"
        sleep 3600
        exit 1
    fi
    if [ -n "$BUNDLE_ID_FILE" ]; then
        if [ -f "$BUNDLE_ID_FILE" ]; then
            echo "$BUNDLE_ID_FILE already exists" 1>&2
            exit 1
        fi
        echo "$bundle_id" > "$BUNDLE_ID_FILE"
    fi
    IFS=':'
    upload_idx=$((upload_idx + 1))
done
IFS="$ifs_orig"

emit_event \
    'uploaddone' \
    'dispatching upload done event'

if [ -z "$SLEEP_INSTEAD_OF_EXIT" ]; then
    exit 0
fi

echo "wrap_datamon sleeping indefinitely (for debug)"
while true; do sleep 100; done
