#!/bin/zsh

setopt ERR_EXIT

TIMESTAMP=$(date '+%y%m%d%H%M%S')
TIMESTAMP_HUMAN_READABLE=$(date '+%Y-%b-%d' | tr '[:upper:]' '[:lower:]')
DATAMON_REPO=backup-filestore-output
DATAMON_CONCURRENCY_FACTOR=300
REMOVE_INTERVAL_DAYS=20

SHELL_LOG_NAME=processing
DATAMON_OUT_LOG_NAME=datamon
DATAMON_ERR_LOG_NAME=datamon_err
BACKUP_FILELIST_NAME=datamover_backup

WD=/filestore/datamover-backup-wd

backup_dir=
backup_dirs_filelist=
unlinkable_filelist=

while getopts d:u:t:f: opt; do
    case $opt in
        (d)
            backup_dir="$OPTARG"
            ;;
        (u)
            unlinkable_filelist="$OPTARG"
            ;;
        (t)
            if [[ "$OPTARG" = 'true' ]]; then
                print -- 'running in test mode'
                DATAMON_REPO=ransom-datamon-test-repo
            else
                if [[ "$OPTARG" != 'false' ]]; then
                    print -- "unexpected test (-t) param $OPTARG" 1>&2
                    exit 1
                fi
            fi
            ;;
        (f)
            backup_dirs_filelist="$OPTARG"
            ;;
        (\?)
            print Bad option, aborting.
            exit 1
            ;;
    esac
done
(( OPTIND > 1 )) && shift $(( OPTIND - 1 ))

if [[ -n $backup_dir && -n $backup_dirs_filelist ]]; then
    print 'backup directory (-d) and backup filelist (-f) params are mutually exclusive' 1>&2
    exit 1
fi
if [[ -z $backup_dir && -z $backup_dirs_filelist ]]; then
    print 'must specify at least one of backup directory (-d) and backup filelist (-f) params' 1>&2
    exit 1
fi
if [[ -n $backup_dir ]]; then
    if [[ ! -d $backup_dir ]]; then
        print "backup directory (-d) $backup_dir doesn't exist" 1>&2
        exit 1
    fi
fi

if [[ ! -d $WD ]]; then
    print "expected working directory $WD not present" 1>&2
    exit
fi

##

rotate_log() {
    typeset log_path_sans_ext
    typeset log_name_sans_ext
    typeset log_dir_path
    typeset last_rot_log
    typeset -i last_rot_log_idx
    log_path_sans_ext="$1"
    log_name_sans_ext=$(basename ${log_path_sans_ext})
    log_dir_path=$(dirname ${log_path_sans_ext})
    if [[ ! -d $log_dir_path ]]; then
        print "log directory $log_dir_path doesn't exist" 1>&2
        exit 1
    fi
    last_rot_log=$(find ${log_dir_path} -maxdepth 1 \
                        -name "${log_name_sans_ext}-*.log" | \
                       sort | \
                       tail -1)
    if [[ ! -e ${log_path_sans_ext}.log ]]; then
        return
    fi
    if [[ ! -f ${log_path_sans_ext}.log ]]; then
        print -- "attempted to rotate ${log_path_sans_ext}.log, not a file" 1>&2
    fi

    if [[ -z $last_rot_log ]]; then
        mv ${log_path_sans_ext}.log ${log_path_sans_ext}-1.log
        else
            last_rot_log_idx=$(print -- $last_rot_log | \
                                   sed "s/.*${log_path_sans_ext}-\(.\+\).log$/\1/")
            if [[ $last_rot_log_idx -eq 0 ]]; then
                print "couldn't detect index for log rotation in $last_rot_log" 1>&2
                exit 1
            fi
            mv ${log_path_sans_ext}.log ${log_path_sans_ext}-$(($last_rot_log_idx + 1)).log
    fi
}

cd $WD

log=${SHELL_LOG_NAME}.log

rotate_log ${SHELL_LOG_NAME}
rotate_log ${DATAMON_OUT_LOG_NAME}
rotate_log ${DATAMON_ERR_LOG_NAME}
if [[ -n $unlinkable_filelist ]]; then
    rotate_log $unlinkable_filelist
fi


##

if [[ ! -d ${HOME}/.datamon ]]; then
    datamon config create \
            --name 'datamover-backerupper' \
            --email 'rwilliams@oneconcern.com'
fi

##

if [[ -z $backup_dirs_filelist ]]; then
    backup_dirs_filelist=/tmp/${BACKUP_FILELIST_NAME}.list
    find $backup_dir -mindepth 1 -maxdepth 1 -type d > $backup_dirs_filelist
fi

typeset -A lineidxs

while read file; do
    print "==Next==" | tee -a $log
    if [[ ! -d $file ]]; then
        print "Skipping ${file}: not a directory" | tee -a $log
        continue
    fi
    line=$(basename $file)
    if [[ -z $lineidxs[$line] ]]; then
        lineidxs[$line]=0
    else
        lineidxs[$line]=$(($lineidxs[$line] + 1))
    fi
    lineidx=$lineidxs[$line]
    date | tee -a $log
    print "Processing ${line} (${lineidx})" | tee -a $log
    # Count number of entries in directory
    find $file -type f |wc -l | tee -a $log
    # Upload to datamon
    label="${TIMESTAMP_HUMAN_READABLE}-${line}-${lineidx}"
    print "label=${label}" | tee -a ${DATAMON_OUT_LOG_NAME}.log | tee -a ${DATAMON_ERR_LOG_NAME}.log
    1>>${DATAMON_OUT_LOG_NAME}.log 2>>${DATAMON_ERR_LOG_NAME}.log \
    datamon bundle upload \
        --concurrency-factor $DATAMON_CONCURRENCY_FACTOR \
        --skip-on-error \
        --repo $DATAMON_REPO \
        --path $file \
        --label $label \
        --message "datamover backup.sh backup: ${TIMESTAMP} (${TIMESTAMP_HUMAN_READABLE})"
    # check number of entries
    datamon bundle list files \
        --repo $DATAMON_REPO \
        --label $label \
        > ${line}-${lineidx}-files.log
    # If correct
    count=$(cat ${line}-${lineidx}-files.log |grep -i '^name:.*, size:.*, hash:.*$' |wc -l)
    print -- "$count in bundle"
    count2=$(find $file -type f |wc -l)
    print -- "$count2 in nfs"
    if [ $count -eq $count2 ]; then
        # confident that current file is backed up
        if [[ -z $unlinkable_filelist ]]; then
            echo "Deleting ${line} (${lineidx})" | tee -a $log
            find $file -mtime "+${REMOVE_INTERVAL_DAYS}" -delete | tee -a $log
        else
            find $file -mtime "+${REMOVE_INTERVAL_DAYS}" >> ${unlinkable_filelist}.log
        fi
    fi
    rm ${line}-${lineidx}-files.log
    echo "Done " $file | tee -a $log
done < $backup_dirs_filelist
