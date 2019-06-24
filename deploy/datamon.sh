#! /bin/sh

### datamon.sh
# push updates to users of github.com/oneconcern/datamon via polling
# of the github releases feature.
#
# keep a cache of releases on the local fs in .datamon/releases/
# according to semantic versioning scheme.  the polling implementation
# relies on a timestamp file that's also in releases/.
#
# additionally, this script verifies binaries against a checksum (currently md5)
# upon download.  these checksums are out-of-band for the github releases
# feature, so cutting a release for use with this script additionally involves generating
# such a checksum file of newline-delimited entries, each of the format
# <checksum> <filename>

## script config options: set these to non-zero to enable
# print debug messages
DEBUG=
# allow users to decide whether to install the latest version with a y/N prompt
CONFIRM_INSTALL=

## escape codes to color messages from this script
# these are used to distinguish debug output from this script from that of datamon itself.
COL_RED='\033[0;31m'
COL_CYAN='\033[0;36m'
COL_NC='\033[0m'

## setup variables used by the script
# github api
REL_URL=https://api.github.com/repos/oneconcern/datamon/releases/latest

# used to map kernel type to binary type
if [ "$(uname)" = "Darwin" ]; then
    archive_tag='mac'
else
    archive_tag='linux'
fi

# root directory used by this script
REL_DIR="$HOME/.datamon/releases"
if [ ! -d "$REL_DIR" ]; then
    mkdir -p "$REL_DIR"
fi

# previous semantic version string (if any)
SEM_VER_PREV=$(cd "$REL_DIR" && find . -maxdepth 1 |grep -v '^.$' |sort |tail -1)
# file used to store timestamps in order to implement poll interval
TIME_FILE="${REL_DIR}/.lastupdate"

# seconds since epoch
CURR_TIME=$(date -j -f "%a %b %d %T %Z %Y" "$(date)" "+%s")
if [ ! -f "$TIME_FILE" ]; then
    echo "$CURR_TIME" > "$TIME_FILE"
fi
LAST_TIME=$(cat "$TIME_FILE")
echo "$CURR_TIME" > "$TIME_FILE"

TIME_DIFF=$(echo "$CURR_TIME - $LAST_TIME" |bc)
POLL_INTERVAL=$(echo '60 * 60 * 24' |bc) # seconds

dbg_print() {
    if [ -n "$DEBUG" ]; then
        echo "${COL_CYAN}$1${COL_NC}"
    fi
}

# emulate zsh (read -q) functionality to implement yes-no prompts
yn_read() {
    echo -n "$1" "[y/N]"
    read -s -n 1 YN_RES
    if [ "$YN_RES" = "Y" ] || [ "$YN_RES" = "y" ]; then
        return 0
    fi
    return 1
}

if [ -z "$SEM_VER_PREV" ] || [ "$TIME_DIFF" -gt "$POLL_INTERVAL" ]; then
    dbg_print 'checking for latest release'
    # the github releases api returns a json string.
    # the hack in this subshell returns the URL to download the latest release..
    TAR_URL=$(curl -s $REL_URL | \
                  grep browser_download_url | \
                  grep "datamon.${archive_tag}.tgz" | \
                  cut -d : -f 2,3 | tr -d \" | tr -d ' ')
    # .. and that URL has the semantic version string within it.
    SEM_VER=$(echo "$TAR_URL" |cut -d / -f 8)
    DL_DIR="${REL_DIR}/${SEM_VER}"
    if [ ! -d "$DL_DIR" ]; then
        mkdir -p "$DL_DIR"
    fi
    if [ ! -f "$DL_DIR/datamon" ]; then
        # the latest version isn't in the local cache
        if [ -z "$SEM_VER_PREV" ] || \
               [ -z "$CONFIRM_INSTALL" ] || \
               yn_read "upgrade from $SEM_VER_PREV to $SEM_VER?"; then
            dbg_print 'getting latest release'
            (cd "$DL_DIR" && \
                 wget "$TAR_URL" && \
                 tar -xvzf "datamon.${archive_tag}.tgz")
            # download the checksum file and verify.
            # todo: test this with multiple releases containing checksum files.
            HASH_URL=$(curl -s $REL_URL | \
                           grep browser_download_url | \
                           grep "datamon.dsc" | \
                           cut -d : -f 2,3 | tr -d \" | tr -d ' ')
            (cd /tmp && \
                 wget "$HASH_URL")
            HASH_EXPECTED=$(grep "${archive_tag}" /tmp/datamon.dsc |cut -d ' ' -f 1)
            HASH_ACTUAL=$(md5sum "$DL_DIR/datamon.${archive_tag}.tgz" |cut -d ' ' -f 1)
            if [ "$HASH_EXPECTED" = "$HASH_ACTUAL" ]; then
                dbg_print 'hashes match'
            else
                echo "hash mismatch" 1>&2
                exit 2
            fi
        else
            dbg_print 'using previous version'
            exec "${REL_DIR}/${SEM_VER_PREV}/datamon" "$@"
        fi
    fi
    exec "$DL_DIR/datamon" "$@"
else
    dbg_print 'using previous version without polling releases'
    exec "${REL_DIR}/${SEM_VER_PREV}/datamon" "$@"
fi
