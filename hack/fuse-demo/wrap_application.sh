#! /bin/sh

### application wrapper (demo)
# half of the container coordination sketch, a script like this one
# is meant to wrap a data-science program running in the main container of an Argo DAG node
# and communciate with a script like wrap_datamon.sh.
#
# coordination in this case starts with the wrap_datamon script

### parse opts

COORD_POINT=

POLL_INTERVAL=1 # sec

SLEEP_INSTEAD_OF_EXIT=

while getopts sc: opt; do
    case $opt in
        (s)
            SLEEP_INSTEAD_OF_EXIT=true
            ;;
        (c)
            COORD_POINT="$OPTARG"
            ;;
        (\?)
            print Bad option, aborting.
            return 1
            ;;
    esac
done
if [ "$OPTIND" -gt 1 ]; then
    shift $(( OPTIND - 1 ))
fi

if [ -z "$COORD_POINT" ]; then
    echo "coordination point not set" 1>&2
    exit 1
fi

### util

# #% =EVENT_NAME= <- wrap_datamon.sh
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

#% wrap_datamon.sh <- =EVENT_NAME=
emit_event() {
    EVENT_NAME="$1"
    DBG_MSG="$2"
    echo "$DBG_MSG"
    touch "${COORD_POINT}/${EVENT_NAME}"
}

### application wrapper

## the following waits on datamon to make a FUSE mount available
await_event \
    'mountdone' \
    'waiting on datamon mount (app wrap)'

## once data is available, the data-science application is started
echo "mount done, executing mock application, '" "$@" "'"

"$@"
app_exit_status="$?"
if [ "$app_exit_status" != "0" ]; then
    echo "application exited with non-zero-status" 1>&2
    exit "$app_exit_status"
fi

echo "mock application done"

## after the application writes its output, notify the sidecar to start uploading it

emit_event \
    'initupload' \
    'dispatching init upload event'

## block until the sidecar finishes uploading
await_event \
    'uploaddone' \
    'waiting on upload'

## COORDINATION ENDS with this container exiting
echo "recved upload done event, exiting"

if [ -z "$SLEEP_INSTEAD_OF_EXIT" ]; then
    exit 0
fi

echo "sleeping indefinitely (for debug)"
while true; do sleep 100; done
