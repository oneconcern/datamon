#! /bin/zsh

setopt ERR_EXIT
setopt PIPE_FAIL

dbg_print() {
    local COL_CYAN
    local COL_RESET
    COL_CYAN=$(tput -Txterm setaf 7)
    COL_RESET=$(tput -Txterm sgr0)
    echo ${COL_CYAN}
    print -- $1
    echo ${COL_RESET}
}

### verify the result of the simulated Argo container coordination
# this is a programmatic test on the results of running the container coordination demo

DATAMON_EXEC=/usr/bin/datamon
COORD_VERIFY_PATH=/tmp/coordverify
DATAMON_REPO=ransom-datamon-test-repo
NS=datamon-ci

# this is a timing taken on the pod, not locally
start_timestamp=$(cat /tmp/datamon_fuse_demo_coord_start_timestamp)

verify_datamon_timestamp() {
    timestamp="$1"
    epoch_timestamp="$(go run ./hack/fuse-demo/parse_timestamp.go "${timestamp}")"
    sec_from_start=$((${epoch_timestamp} - ${start_timestamp}))
    #
    dbg_print "timestamp ${timestamp} => ${epoch_timestamp}"
    dbg_print "start_timestamp ${start_timestamp}"
    dbg_print "timestamp_to_parse $timestamp_to_parse epoch_timestamp $epoch_timestamp -- sec_from_start ${sec_from_start}"
    if [ ! "$sec_from_start" -gt 0 ]; then
        print -- ${sec_from_start}
        echo 'label timestamp not after demo start' 1>&2
        exit 1
    fi
}

EXPECTED_LABEL=coordemo
dbg_print 'getting hash from label'

export DATAMON_GLOBAL_CONFIG='datamon-config-test-sdjfhga'

params_label_get=(--repo "$DATAMON_REPO" \
                         --label "$EXPECTED_LABEL" \
                         --context 'datamon-sidecar-test')

dbg_print 'label get params'
dbg_print '----'
print -l -- $params_label_get
dbg_print '----'
dbg_print 'label get lines'
"$DATAMON_EXEC" label get \
                ${params_label_get}
dbg_print '==============='

label_get_line=$("$DATAMON_EXEC" label get \
                      $params_label_get | \
                      tail -1)
HASH_FROM_LABEL=$(echo "$label_get_line" |cut -d"," -f 2 |tr -d ' ')
verify_datamon_timestamp "$(echo "$label_get_line" |cut -d"," -f 3 |sed 's/^ *//')"

if [ -z "HASH_FROM_LABEL" ]; then
    echo "didn't find expected label $EXPECTED_LABEL" 1>&2
    exit 1
fi

if [ -d "$COORD_VERIFY_PATH" ]; then
    rm -rf "$COORD_VERIFY_PATH"
fi

mkdir "$COORD_VERIFY_PATH"


HASH_TO_DOWNLOAD=$HASH_FROM_LABEL

##
# verify the bundle id file written according to the `-i` parameter to wrap_datamon.sh
dbg_print 'getting hash from bundle id file'

BUNDLE_ID_FILE=bundleid.txt
if [[ -e $BUNDLE_ID_FILE ]]; then
    print 'removing stale bundleid file'
    rm $BUNDLE_ID_FILE
fi
SIDECAR_TAG=$(go run ./hack/release_tag.go)
pod_name=$(kubectl -n $NS  get pods -l app=datamon-coord-fuse-demo,instance="${SIDECAR_TAG}" | grep Running | sed 's/ .*//')
kubectl -n $NS cp $pod_name:/tmp/bundleid.txt $BUNDLE_ID_FILE -c datamon-sidecar
HASH_FROM_SIDECAR_OUTPUT=$(cat $BUNDLE_ID_FILE | tr -d ' ')
rm $BUNDLE_ID_FILE

##

if [ -z "$HASH_TO_DOWNLOAD" ]; then
    echo 'hash to download unset' 2>&1
    exit 1
fi

if [ "$HASH_TO_DOWNLOAD" != "$HASH_FROM_LABEL" ]; then
    echo "message hash doesn't match label hash" 1>&2
    exit 1
fi

if [ "$HASH_TO_DOWNLOAD" != "$HASH_FROM_SIDECAR_OUTPUT" ]; then
    echo "message hash doesn't match sidecar output hash" 1>&2
    exit 1
fi

dbg_print 'downloading bundle'


params_bundle_download=(--repo "$DATAMON_REPO" \
                               --destination "$COORD_VERIFY_PATH" \
                               --bundle "$HASH_TO_DOWNLOAD" \
                               --context 'datamon-sidecar-test')
"$DATAMON_EXEC" \
    bundle download \
    $params_bundle_download

if [ ! -f "$COORD_VERIFY_PATH/result" ]; then
    echo "didn't find result file $COORD_VERIFY_PATH/result" 1>&2
    exit 1
else
    dbg_print "found result file '$COORD_VERIFY_PATH/result'"
    cat "$COORD_VERIFY_PATH/result"
    dbg_print '===='
fi

num_files=$(cat "$COORD_VERIFY_PATH/result" |cut -d"," -f 1)
num_lines_first_file=$(cat "$COORD_VERIFY_PATH/result" |cut -d"," -f 2)
first_file_name=$(cat "$COORD_VERIFY_PATH/result" |cut -d"," -f 3)

if [ "$num_files" = "4" ]; then
    echo 'found expected file count in result'
else
    1>&2 print -- "${num_files} != 4"
    echo 'unexpected number of files in result' 1>&2
    exit 1
fi

if [ "$num_lines_first_file" = "45" ]; then
    echo 'found expected line count in result'
else
    1>&2 print -- "${num_lines_first_file} != 45"
    echo "unexpected number of lines in result $num_lines_first_file" 1>&2
    exit 1
fi
