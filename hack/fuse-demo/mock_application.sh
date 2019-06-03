#! /bin/sh

### data-science application placeholder/mock
# this program is a stand-in for a data-science application:
# it reads some data out of datamon, performs some calculations,
# and outputs the result to datamon to upload

MOUNT_POINT="$1"
UPLOAD_SOURCE="$2"

if [ -z "$MOUNT_POINT" ]; then
    echo 'usage: ./mock_application.sh mount-point upload-source' 1>&2
    exit 1
fi
if [ -z "$UPLOAD_SOURCE" ]; then
    echo 'usage: ./mock_application.sh mount-point upload-source' 1>&2
    exit 1
fi

num_files=$(ls -1 "$MOUNT_POINT" |wc -l)

first_file=$(ls -1 "$MOUNT_POINT" |head -1)

num_lines_first_file=$(wc -l "$MOUNT_POINT/$first_file" |tr -s ' ' |cut -d" " -f 1)

echo "$num_files,$num_lines_first_file" > "$UPLOAD_SOURCE/result"
