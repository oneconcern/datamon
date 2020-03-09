#! /bin/sh
#
# An application wrapper to coordinate data retrieval and save workflows with datamon sidecars.
#
# This is meant to wrap a data-science program running in the main container of an Argo DAG node
# and communicate with a script like wrap_datamon.sh.

COORD_POINT=
POLL_INTERVAL=1 # sec
SLEEP_INSTEAD_OF_EXIT=

SC_FUSE=
SC_PG=
WRONG_PARAM=
DB_LIST=
while getopts sc:b:d: opt; do
    case $opt in
        (s)
            SLEEP_INSTEAD_OF_EXIT=true
            ;;
        (c)
            COORD_POINT="$OPTARG"
            ;;
        (b)
            battery_type="$OPTARG"
            if [ "$battery_type" = "fuse" ]; then
                WRONG_PARAM="false"
                SC_FUSE=true
            elif [ "$battery_type" = "postgres" ]; then
                WRONG_PARAM="false"
                SC_PG=true
            else
                echo "unkown battery type $battery_type" 1>&2
                exit 1
            fi
            ;;
        (d)
            DB_LIST="$OPTARG"
            ;;
        (\?)
            echo "Usage: wrap_application.sh [-s] -b {fuse|postgres} -c {path}"
            exit 1
            ;;
    esac
done
if [ "$OPTIND" -gt 1 ]; then
    shift $(( OPTIND - 1 ))
fi

if [ -z "$COORD_POINT" ]; then
    echo "coordination point not set"  1>&2
    exit 1
fi

if [ -z "$WRONG_PARAM" ]; then
  echo "Datamon Wrap Application Script: Set type of mount (-b) to be fuse or postgres" 1>&2
  exit 1
fi

if [ "$SC_PG" = "true" ] ; then
  if [ -z "$DB_LIST" ] ; then
    echo "When using the Wrap Application script to coordinate with postgres sidecars, you must specify the list of databases waited for"
    exit 1
  fi
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

# #% =EVENT_NAME= <- wrap_datamon.sh
await_pg_event() {
    COORD_DONE=
    EVENT_NAME="$1"
    DB="$2"
    DBG_MSG="$3"
    DBG_POLLS="$4"
    if [ -n "$DBG_MSG" ]; then
        echo "$DBG_MSG"
    fi
    if [ ! -d "${COORD_POINT}/${DB}" ] ; then
      mkdir -p "${COORD_POINT}/${DB}"
    fi
    while [ -z "$COORD_DONE" ]; do
        if [ -f "${COORD_POINT}/${DB}/${EVENT_NAME}" ]; then
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

emit_pg_event() {
    EVENT_NAME="$1"
    DB="$2"
    DBG_MSG="$3"
    echo "$DBG_MSG"
    if [ ! -d "${COORD_POINT}/${DB}" ] ; then
      mkdir -p "${COORD_POINT}/${DB}"
    fi
    touch "${COORD_POINT}/${DB}/${EVENT_NAME}"
}

### application wrapper

## the following waits on datamon to make a FUSE mount available
if [ -n "$SC_FUSE" ]; then
    await_event \
        'mountdone' \
        'waiting on datamon mount (app wrap)'
fi
if [ -n "$SC_PG" ]; then
  for db in ${DB_LIST} ; do
    await_pg_event \
        'dbstarted' \
        "${db}" \
        'waiting on db start (app wrap)' \
        'dispatching init db upload event'
  done
fi

## once data is available, the data-science application is started
echo "mount done, executing main application, '" "$@" "'"

"$@"
app_exit_status="$?"
if [ "$app_exit_status" != "0" ]; then
    echo "application exited with non-zero-status" 1>&2
    exit "$app_exit_status"
fi

echo "main application done"

## after the application writes its output, notify the sidecar to start uploading it
if [ -n "$SC_FUSE" ]; then
    emit_event \
        'initupload' \
        'dispatching init upload event'
fi
if [ -n "$SC_PG" ]; then
  for db in ${DB_LIST} ; do
    emit_pg_event \
        'initdbupload' \
        "${db}" \
        'dispatching init db upload event'
  done
fi

## block until the sidecar finishes uploading
if [ -n "$SC_FUSE" ]; then
    await_event \
        'uploaddone' \
        'waiting on upload'
fi
if [ -n "$SC_PG" ]; then
  for db in ${DB_LIST} ; do
    await_pg_event \
        'dbuploaddone' \
        "${db}" \
        'waiting on db upload'
  done
fi

## COORDINATION ENDS with this container exiting
echo "received upload done event, exiting"

if [ -z "$SLEEP_INSTEAD_OF_EXIT" ]; then
    exit 0
fi

echo "wrap_application sleeping indefinitely (for debug)"
SLEEP_TIMEOUT=600
timeout "${SLEEP_TIMEOUT}s" sh -c 'while true;do echo "zzzz...." && sleep 5;done' || true
