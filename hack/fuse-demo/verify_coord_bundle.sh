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
HASH_FROM_LABEL=

"$DATAMON_EXEC" label list --repo "$DATAMON_REPO" 2>&1 | \
    tail +2 | \
    while read label_list_line; do
        label=$(echo "$label_list_line" |cut -d"," -f 1 |tr -d ' ')
        hash=$(echo "$label_list_line" |cut -d"," -f 2 |tr -d ' ')
        timestamp=$(echo "$label_list_line" |cut -d"," -f 3 |sed 's/^ *//')
        if [ "$EXPECTED_LABEL" = "$label" ]; then
            HASH_FROM_LABEL="$hash"
            verify_datamon_timestamp "$timestamp"
            break
        fi
    done

# todo: variable scoping within above `while read` loop

if [ -z "HASH_FROM_LABEL" ]; then
    echo "didn't find expected label $EXPECTED_LABEL" 1>&2
    exit 1
fi

if [ -d "$COORD_VERIFY_PATH" ]; then
    rm -rf "$COORD_VERIFY_PATH"
fi

mkdir "$COORD_VERIFY_PATH"

# todo: set hash to download from label after bugfix
HASH_TO_DOWNLOAD=

most_recent_bundle_list_entry=$("$DATAMON_EXEC" bundle list \
                                                --repo ransom-datamon-test-repo 2>&1 | \
    grep 'container coordination demo$' | \
    tail -1)
HASH_TO_DOWNLOAD=$(echo "$most_recent_bundle_list_entry" | cut -d"," -f 1 |tr -d ' ')
bundle_list_entry_timestamp=$(echo "$most_recent_bundle_list_entry" | cut -d"," -f 2 \
                                  |sed 's/^ *//' |sed 's/ *$//')

verify_datamon_timestamp "$bundle_list_entry_timestamp"

if [ "$HASH_TO_DOWNLOAD" != "$HASH_FROM_LABEL" ]; then
    echo "message hash doesn't match label hash" 1>&2
    exit 1
fi

if [ -z "$HASH_TO_DOWNLOAD" ]; then
    echo 'hash to download unset' 2>&1
    exit 1
fi

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
