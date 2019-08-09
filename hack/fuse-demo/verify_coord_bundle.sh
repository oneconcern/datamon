#! /bin/zsh

### verify the result of the simulated Argo container coordination
# this is a programmatic test on the results of running the container coordiation demo

DATAMON_EXEC=./cmd/datamon/datamon
COORD_VERIFY_PATH=/tmp/coordverify
DATAMON_REPO=ransom-datamon-test-repo

start_timestamp=$(cat /tmp/datamon_fuse_demo_coord_start_timestamp)

verify_datamon_timestamp() {
    timestamp="$1"
    timestamp_to_parse=$(echo "$timestamp" | sed 's/\.[^ ]*//g' | sed 's/ *[^ ]*$//')
    epoch_timestamp=$(date -jf '%Y-%m-%d %H:%M:%S %z' "$timestamp_to_parse" '+%s')
    sec_from_start=$(echo "$epoch_timestamp - $start_timestamp" |bc)

    #
    echo "timestamp_to_parse $timestamp_to_parse"

    if [ ! "$sec_from_start" -gt 0 ]; then
        echo 'label timestamp not after demo start' 1>&2
        exit 1
    fi
}

EXPECTED_LABEL=coordemo
print 'getting hash from label'

label_list_line=$(2>&1 "$DATAMON_EXEC" label get \
                       --repo "$DATAMON_REPO" \
                       --label $EXPECTED_LABEL | \
                      tail -1)
HASH_FROM_LABEL=$(echo "$label_list_line" |cut -d"," -f 2 |tr -d ' ')
verify_datamon_timestamp "$(echo "$label_list_line" |cut -d"," -f 3 |sed 's/^ *//')"

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
print 'getting hash from bundle id file'

BUNDLE_ID_FILE=bundleid.txt
if [[ -e $BUNDLE_ID_FILE ]]; then
    print 'removing stale bundleid file'
    rm $BUNDLE_ID_FILE
fi
pod_name=$(kubectl get pods -l app=datamon-coord-demo | grep Running | sed 's/ .*//')
kubectl cp $pod_name:/tmp/bundleid.txt $BUNDLE_ID_FILE -c datamon-sidecar
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

print 'downloading bundle'

"$DATAMON_EXEC" \
    bundle download \
    --repo "$DATAMON_REPO" \
    --destination "$COORD_VERIFY_PATH" \
    --bundle "$HASH_TO_DOWNLOAD"

if [ ! -f "$COORD_VERIFY_PATH/result" ]; then
    echo "didn't find result file $COORD_VERIFY_PATH/result" 1>&2
    exit 1
fi

num_files=$(cat "$COORD_VERIFY_PATH/result" |cut -d"," -f 1)
num_lines_first_file=$(cat "$COORD_VERIFY_PATH/result" |cut -d"," -f 2)

if [ "$num_files" = "5" ]; then
    echo 'found expected file count in result'
else
    echo 'unexpected number of files in result' 1>&2
    exit 1
fi

if [ "$num_lines_first_file" = "32" ]; then
    echo 'found expected line count in result'
else
    echo "unexpected number of lines in result $num_lines_first_file" 1>&2
    exit 1
fi
