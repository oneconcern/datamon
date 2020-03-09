#! /bin/bash
#
# A sidecar to coordinate datamon actions with postgres
#
typeset COL_GREEN=$(tput -Txterm setaf 2)
typeset COL_CYAN=$(tput -Txterm setaf 14)
typeset COL_RED=$(tput -Txterm setaf 9)
typeset COL_RESET=$(tput -Txterm sgr0)
typeset dbg=true

dbg_print() {
    if [[ "${dbg}" != "true" ]] ; then
      return
    fi
    echo "${COL_CYAN}DEBUG: $*${COL_RESET}"
}

info_print() {
    echo "${COL_GREEN}INFO: $*${COL_RESET}"
}

error_print() {
    echo "${COL_RED}ERROR: $*${COL_RESET}"
}

terminate() {
    error_print "$*"
    exit 1
}

parse_yaml() {
   local prefix=$2
   local s='[[:space:]]*'
   w='[a-zA-Z0-9_]*'
   fs=$(echo @|tr @ '\034')
   # shellcheck disable=SC1117
   sed -ne "s|^\($s\)\($w\)$s:$s\"\(.*\)\"$s\$|\1$fs\2$fs\3|p" \
        -e "s|^\($s\)\($w\)$s:$s\(.*\)$s\$|\1$fs\2$fs\3|p"  "$1" |
   awk -F"$fs" '{
      indent = length($1)/2;
      vname[indent] = $2;
      for (i in vname) {if (i > indent) {delete vname[i]}}
      if (length($3) > 0) {
         vn=""; for (i=0; i<indent; i++) {vn=(vn)(vname[i])("_")}
         printf("export %s%s%s=\"%s\"\n", "'"${prefix}"'",toupper(vn), toupper($2), $3);
      }
   }'
}

LOG_ROOT=/tmp/sidecar-logs
PG_VERSION=$(pg_config --version | sed -E 's/^(\w+)\s+([0-9\.]+)\s.*/\2/')
PG_SU=postgres
USER=$(id -u)

mkdir -p ${LOG_ROOT}

info_print "running pg sidecar as user: $USER"
info_print "running pg version: $PG_VERSION"

# Parameters from config (override any env var)
SIDECAR_CONFIG=${SIDECAR_CONFIG:-"/config/pgparams.yaml"}
if [[ -f "${SIDECAR_CONFIG}" ]] ; then
  ##dbg_print "sidecar config: $(cat "${SIDECAR_CONFIG}")"
  eval "$(parse_yaml "${SIDECAR_CONFIG}" "SIDECAR_")"
fi
## dbg_print "sidecar ENV: $(env)"

# Pick parameters from resolved env (initial env+config)
POLL_INTERVAL="${SIDECAR_GLOBALOPTS_POLLINTERVAL:-"1"}" # sec

COORD_POINT="${SIDECAR_GLOBALOPTS_COORDPOINT:-"/tmp/coord"}"
SLEEP_INSTEAD_OF_EXIT="${SIDECAR_GLOBALOPTS_SLEEPINSTEADOFEXIT:-"false"}"
SLEEP_TIMEOUT="${SIDECAR_GLOBALOPTS_SLEEPTIMEOUT:-"600"}" # sec
PG_DATA_DIR_ROOT="${SIDECAR_DATABASE_DATADIR:-"/pg_stage"}"
PG_DB_ID="${SIDECAR_DATABASE_NAME:-"db"}"
PG_DB_PORT="${SIDECAR_DATABASE_PGPORT:-"5432"}"

PG_DB_SRCREPO="${SIDECAR_DATABASE_SRCREPO}"
PG_DB_SRCLABEL="${SIDECAR_DATABASE_SRCLABEL}"
PG_DB_SRCBUNDLE="${SIDECAR_DATABASE_SRCBUNDLE}"

PG_DB_DESTREPO="${SIDECAR_DATABASE_DESTREPO}"
PG_DB_DESTLABEL="${SIDECAR_DATABASE_DESTLABEL}"
PG_DB_DESTMSG="${SIDECAR_DATABASE_DESTMESSAGE}"

PG_OWNER="${SIDECAR_DATABASE_OWNER:-$PG_SU}"

data_dir="${PG_DATA_DIR_ROOT}/${PG_DB_ID}"
if [[ -e "${data_dir}" ]] ; then
  if ! sudo rm -rf "${data_dir}" ; then
    terminate "failed to clean ${data_dir} before starting"
  fi
fi
sudo mkdir -p "${data_dir}" && sudo chown "$USER" "${data_dir}" && chmod 750 "${data_dir}"

# this special file is used to recognize the postgres version the bundle was created with
version_path=${data_dir}/pg_version

# default config is baked on the container image. This may be overriden by mounting
# /home/developer/.datamon2/datamon.yaml

# #% =EVENT_NAME= <- wrap_application.sh
await_pg_event() {
    COORD_DONE=
    EVENT_NAME="$1"
    DB="$2"
    DBG_MSG="$3"
    DBG_POLLS="$4"
    if [[ ! -d "${COORD_POINT}/${DB}" ]] ; then
      mkdir -p "${COORD_POINT}/${DB}"
    fi
    if [[ -n "${DBG_MSG}" ]]; then
        dbg_print "$DBG_MSG"
    fi
    while [[ -z "${COORD_DONE}" ]]; do
        if [[ -f "${COORD_POINT}/${DB}/${EVENT_NAME}" ]]; then
            COORD_DONE=1
        fi
        if [[ -n "${DBG_POLLS}" ]]; then
            dbg_print "... ${DBG_MSG} ..."
        fi
        sleep "${POLL_INTERVAL}"
    done
}

#% wrap_application.sh <- =EVENT_NAME=
emit_pg_event() {
    EVENT_NAME="$1"
    DB="$2"
    DBG_MSG="$3"
    dbg_print "${DBG_MSG}"
    if [[ ! -d "${COORD_POINT}/${DB}" ]] ; then
      mkdir -p "${COORD_POINT}/${DB}"
    fi
    touch "${COORD_POINT}/${DB}/${EVENT_NAME}"
}

# parameters validation
if [[ -n "${PG_DB_DESTREPO}" && -z "${PG_DB_DESTMSG}" ]]; then
   terminate "missing commit message to use when saving db $PG_DB_ID"
fi
if [[ -n "${PG_DB_SRCREPO}" && -n "${PG_DB_SRCBUNDLE}" && -n "${PG_DB_SRCLABEL}" ]]; then
        terminate "specifying source data by bundleID or label is mutually exclusive"
fi
if [[ -z "${PG_DB_DESTREPO}" ]]; then
   info_print "no destination repo is specified: this db instance will not be saved to datamon"
fi
if [[ -z "${PG_DB_SRCREPO}" ]]; then
   info_print "no source repo is specified: a new instance will be created"
fi
if [[ -z "${PG_DB_SRCREPO}" && -z "${PG_DB_DESTREPO}" ]] ; then
   info_print "neither source repo nor dest repo specified: a new instance will be created but not saved. Kinda useless: you might want to check your config"
fi

if [[ -n "${PG_DB_SRCREPO}" ]]; then
    # a source has been specified: download the database
    params=(bundle download --repo "${PG_DB_SRCREPO}" --destination "${data_dir}")
    if   [[ -n "${PG_DB_SRCLABEL}" ]]; then
        params+=(--label "${PG_DB_SRCLABEL}")
    elif [[ -n "${PG_DB_SRCBUNDLE}" ]]; then
        params+=(--bundle "${PG_DB_SRCBUNDLE}")
    else
        info_print "retrieving latest bundle for repo ${PG_DB_SRCREPO}"
    fi
    log_download="${LOG_ROOT}/datamon_download.${PG_DB_ID}.log"
    info_print "retrieving database from datamon repo ${PG_DB_SRCREPO}"
    info_print "datamon ${params[*]}"
    info_print "with log ${log_download}"
    if ! datamon "${params[@]}" > "${log_download}" 2>&1  ; then
        cat "${log_download}"
        terminate "error starting \"datamon ${params[*]}\""
    fi
    info_print "database files downloaded"
    # restore well-known static directory structure required by pg (baked on image)
    (cd "${data_dir}" && tar xf ~/pgdirs.tar)
    # abide by pg requirements on mode
    chmod -R o-rwx "${data_dir}"

    # verify metadata about pg version (only major versions matter: x.y)
    bundle_pg_version=""
    if [[ -f "${version_path}" ]]; then
        bundle_pg_version=$(cat "${version_path}")
    else
        terminate "didn't find version file at ${version_path}"
    fi
    bundle_major_version=$(echo "${bundle_pg_version}"|cut -d'.' -f1)
    pg_major_version=$(echo "${PG_VERSION}"|cut -d'.' -f1)
    if [[ "${bundle_major_version}" != "${pg_major_version}" ]]; then
        terminate "pg major version mistmatch: found bundle created with: $bundle_pg_version -- using: $PG_VERSION"
    fi
else
    # no source has been specified: create a new empty database server (no database is created but the default templates)
    info_print "initializing a new database server: initdb --no-locale --encoding UTF8 -D ${data_dir}"
    initdb --no-locale --encoding UTF8 -D "${data_dir}" -U ${PG_SU}
    echo "${PG_VERSION}" > "${version_path}"
fi

info_print "starting postgres database processes"
log_pg="${LOG_ROOT}/pg.${PG_DB_ID}.log"
touch "${log_pg}"
info_print "starting postgres: pg_ctl -D $data_dir -p ${PG_DB_PORT} -l ${log_pg} start"
if ! pg_ctl -D "${data_dir}" --options "-p ${PG_DB_PORT}" --log "${log_pg}" start ; then
    cat "${log_pg}"
    terminate "error starting postgres"
fi

if [[ -z "${PG_DB_SRCREPO}" && "${PG_OWNER}" != "${PG_SU}" ]]; then
    dbg_print "db start with ${PG_DB_ID}: createuser ${PG_OWNER}"
    if ! createuser -p "${PG_DB_PORT}" -s -U ${PG_SU} "${PG_OWNER}" ; then
        terminate "user creation in newly created db ${PG_DB_ID} failed"
    fi
fi

info_print "postgres database started"

emit_pg_event \
    'dbstarted' \
    "${PG_DB_ID}" \
    'dispatching db started event'

await_pg_event \
    'initdbupload' \
    "${PG_DB_ID}" \
    'waiting on db upload event'

# VACUUM is a Postgres-specific SQL addition
# that allows the filesystem representation of the database
# to take less disk usage.
admin_args=(-h localhost -p "${PG_DB_PORT}" -U "${PG_SU}")
databases=$(psql "${admin_args[@]}"  --list -t -x -A|grep Name|cut -d'|' -f2)
dbg_print "databases on server ${PG_DB_ID}: $(echo "${databases}"|tr '\n' ' ')"
info_print "bouncing connections..."
echo "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname!='postgres';" | psql "${admin_args[@]}"
info_print "quiescing postgres databases..."
for db in ${databases} ; do
    if [[ "${db}" == "postgres" || ${db} =~ ^template[0-9]+ ]] ; then
      continue
    fi
    db_args=("${admin_args[@]}" "${db}")
    echo 'VACUUM;' | psql "${db_args[@]}"
    echo 'CHECKPOINT;' | psql "${db_args[@]}"
done
info_print "stopping postgres ${PG_DB_ID}"
pg_ctl -D "${data_dir}" -m immediate stop

if [[ -z "${PG_DB_DESTREPO}" ]] ; then
    info_print "Read-Only database: ${PG_DB_ID} is not saved to datamon"
else
    info_print "uploading postgres database to datamon repo ${PG_DB_DESTREPO}"
    log_upload="${LOG_ROOT}/datamon_upload.${PG_DB_ID}.log"
    upload_params=(bundle upload --path "${data_dir}" --message "${PG_DB_DESTMSG}" --repo "${PG_DB_DESTREPO}")
    if [[ -n "${PG_DB_DESTLABEL}" ]]; then
        upload_params+=(--label "${PG_DB_DESTLABEL}")
    fi

    info_print "datamon ${upload_params[*]}"
    info_print "with log file at ${log_upload}"
    if ! datamon "${upload_params[@]}" ; then
        terminate "upload failed"
    fi
    info_print "upload completed for ${PG_DB_ID}"
fi

info_print "signaling upload is done to application"
emit_pg_event \
    'dbuploaddone' \
    "${PG_DB_ID}" \
    'dispatching db upload done event'

# regular operating mode: sidecar exits once done
if [[  "${SLEEP_INSTEAD_OF_EXIT}" != "true" ]] ; then
    exit 0
fi

# debug mode: sleeps for a while before exiting, so a dev has a chance to take a look (defaults to 5m, configurable)
dbg_print "wrap_datamon_pg sleeping for ${SLEEP_TIMEOUT} secs before exiting (for debug)"
dbg_print "wrap_datamon_pg sleeping indefinitely"
timeout "${SLEEP_TIMEOUT}s" bash -c 'while true;do echo "zzzz...." && sleep 5;done' || true
