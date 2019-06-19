#! /bin/zsh

CMD_LOG='cmd.log'
TIME_LOG='time.log'

while getopts l:t: opt; do
    case $opt in
        (t)
            TIME_LOG=$OPTARG
            ;;
        (l)
            CMD_LOG=$OPTARG
            ;;
        (\?)
            print Bad option, aborting.
            return 1
            ;;
    esac
done
(( OPTIND > 1 )) && shift $(( OPTIND - 1 ))

1>"$CMD_LOG" 2>"$TIME_LOG" /usr/bin/time -lp "$@"

echo "wrote '$@' system profiling to $TIME_LOG, command output to $CMD_LOG"
