#! /bin/zsh

setopt ERR_EXIT

LOG_PATH=/tmp/datamover_metrics.log

DM_BIN=datamover

TEST_DIR=/mnt/shared/datamover-test

tot_size_tb=1
num_files=100000

write_files=true
write_files_only=false

while getopts d:s:n:o:w opt; do
    case $opt in
        (o)
            write_files_only="$OPTARG"
            ;;
        (w)
            write_files=false
            ;;
        (d)
            TEST_DIR="$OPTARG"
            ;;
        (s)
            tot_size_tb="$OPTARG"
            ;;
        (n)
            num_files="$OPTARG"
            ;;
        (\?)
            print Bad option, aborting.
            exit 1
            ;;
    esac
done
(( OPTIND > 1 )) && shift $(( OPTIND - 1 ))

if [[ ! ($write_files_only = true || $write_files_only = false) ]]; then
    print "write files only -o flag must be set to either 'true' or 'false'." \
          "got '$write_files_only'" 1>&2
    exit 1
fi

tot_size_mib=$(($tot_size_tb * 1000 * 1000))
filesize_mib=$(print "scale=10; $tot_size_mib / $num_files" |bc)

if [[ -d $TEST_DIR ]]; then
    print "unlinking extant test directory $TEST_DIR"
    rm -rf $TEST_DIR
fi

cd $HOME

print "tot_size_tb: $tot_size_tb"
print "filesize_mib: $filesize_mib"
print "num_files: $num_files"
print "tot_size_tb: $tot_size_tb" >> $LOG_PATH
print "filesize_mib: $filesize_mib" >> $LOG_PATH
print "num_files: $num_files" >> $LOG_PATH

if $write_files; then
    print 'writing files to share'
    mkdir $TEST_DIR
    WRITE_TIMESTAMP_A=$(date +%y%m%d%H%M%S)
    sleep 2
    datamon_metrics write-files \
                    --filesize $filesize_mib \
                    --num-files $(($num_files / 2)) \
                    --out $TEST_DIR/sample_a-${WRITE_TIMESTAMP_A}
    sleep 3
    WRITE_TIMESTAMP_B=$(date +%y%m%d%H%M%S)
    sleep 2
    datamon_metrics write-files \
                    --filesize $filesize_mib \
                    --num-files $(($num_files / 2)) \
                    --out $TEST_DIR/sample_b-${WRITE_TIMESTAMP_B}
fi

if $write_files_only; then
    while true; do sleep 100; done
fi

typeset -A timer_state

stop_timer() {
    typeset timer_name=$timer_state[name]
    typeset timer_start=$timer_state[start]
    typeset timer_end=$(date +%y%m%d%H%M%S)
    typeset log_msg="$timer_name end-time: $timer_end"
    print $log_msg
    print $log_msg >> $LOG_PATH
    timer_state=()
}

start_timer() {
    if [[ ! $#timer_state -eq 0 ]]; then
        stop_timer
    fi
    typeset timer_name=$1
    typeset timer_start=$(date +%y%m%d%H%M%S)
    typeset log_msg="$timer_name start-time: $timer_start"
    print $log_msg
    print $log_msg >> $LOG_PATH
    timer_state[name]=$timer_name
    timer_state[start]=$timer_start
}

BUNDLE_LABEL="datamover-$(date +%y%m%d%H%M%S)"

print 'running datamover'

start_timer datamover

$DM_BIN \
    -d $TEST_DIR \
    -t $WRITE_TIMESTAMP_B \
    -l $BUNDLE_LABEL

stop_timer

## semi-fragile smoke test.  could remove or shore-up.
tot_rm_lines=$(cat /tmp/removable.list \
                   |wc -l |tr -s ' ' |cut -d' ' -f 2)
sample_a_rm_lines=$(cat /tmp/removable.list | grep sample_a \
                   |wc -l |tr -s ' ' |cut -d' ' -f 2)
if [[ ! $tot_rm_lines -eq $sample_a_rm_lines ]]; then
    print 'expected all removal list lines to be files in sample A' 1>&2
    exit 1
fi
if [[ ! $tot_rm_lines -eq $(($num_files / 2)) ]]; then
    print 'expected removal list lines to comprise half the number of files' 1>&2
    exit 1
fi

####

print 'unlinking test directory'
rm -rf $TEST_DIR
print 'creating download directory'
mkdir $TEST_DIR

start_timer download

datamon bundle download \
        --repo ransom-datamon-test-repo \
        --label $BUNDLE_LABEL \
        --concurrency-factor 400 \
        --destination $TEST_DIR

stop_timer


# infinite sleep to debug, gather logs
while true; do sleep 100; done
