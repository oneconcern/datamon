#! /bin/zsh

print -- '### data-science application placeholder/mock
# this program is a stand-in for a data-science application:
# it reads some data out of datamon, performs some calculations,
# and outputs the result to datamon to upload
#
#
#'

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

dbg_print() {
    msg=$1
    print -- $msg
}

# dbg prints

dbg_print '===='
dbg_print "files in mount point ${MOUNT_POINT}"
dbg_print '----'
print -l -- $(ls ${MOUNT_POINT})
dbg_print '===='

typeset -i num_files
num_files=$(ls -1 "$MOUNT_POINT" |wc -l)

typeset first_file
typeset -i num_lines_first_file
if [[ $num_files -gt 0 ]]; then
    dbg_print 'found some files in the output'
    first_file=$(ls -1 "$MOUNT_POINT" |head -1)
    if [[ ! -f $first_file ]]; then
        dbg_print 'first file the list a directory.'
        fidx=4
        dbg_print "search idx ${fidx} file in ${first_file} ..."
        first_file=$(find $MOUNT_POINT -type f | \
                         head -${fidx} | tail -1 | \
                         sed 's/^..//' | \
                         sed 's@^mp/mount@@')
        dbg_print "... found file ${first_file}"
    fi
    if [[ ! -f ${MOUNT_POINT}/${first_file} ]]; then
        1>&2 print -- "didn't find file in ${MOUNT_POINT}"
        exit 1
    fi
    dbg_print "counting lines in ${MOUNT_POINT}/${first_file}"
    dbg_print '----'
    cat ${MOUNT_POINT}/${first_file}
    dbg_print '===='
    num_lines_first_file=$(cat ${MOUNT_POINT}/${first_file} | \
                               wc -l | \
                               tr -s ' ' | cut -d" " -f 1)
    dbg_print "counted ${num_lines_first_file} lines"
else
    dbg_print '**no files at mount point**'
fi

echo "$num_files,$num_lines_first_file,$first_file" \
     > "$UPLOAD_SOURCE/result"

print -- '#
#
#'
