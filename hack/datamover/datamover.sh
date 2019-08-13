#! /bin/zsh

# use the ZERR trap to finalize (e.g. set different exit code, etc.)
setopt ERR_EXIT

typeset -a dirs

TIMESTAMP=$(date +%y%m%d%H%M%S)

BUNDLE_LABEL="datamover-${TIMESTAMP}"

# todo: set to 'backup-filestore-output' before shipping
REPO=ransom-datamon-test-repo

typeset -i concurrency_factor
concurrency_factor=200

unlink=false
# before last NFS bkup.. todo: verify whether this is accurate
timestamp_filter_before=090725000000

filelist_dir=/tmp

while getopts d:c:l:ut:f: opt; do
    case $opt in
        (l)
            BUNDLE_LABEL="$OPTARG"
            ;;
        (d)
            dirs=("$OPTARG" $dirs)
            ;;
        (c)
            concurrency_factor="$OPTARG"
            ;;
        (u)
            unlink=true
            ;;
        (t)
            timestamp_filter_before="$OPTARG"
            ;;
        (f)
            filelist_dir="$OPTARG"
            ;;
        (\?)
            print Bad option, aborting.
            exit 1
            ;;
    esac
done
(( OPTIND > 1 )) && shift $(( OPTIND - 1 ))

print "ensuring filelist directory $filelist_dir"
if [[ ! -d $filelist_dir ]]; then
    mkdir -p $filelist_dir
fi

UPLOAD_FILELIST=${filelist_dir}/upload.list
UPLOADED_FILELIST=${filelist_dir}/uploaded.list
REMOVABLE_FILELIST=${filelist_dir}/removable.list

if [[ -e $UPLOAD_FILELIST ]]; then
    print "$UPLOAD_FILELIST already exists" 1>&2
    exit 1
fi
if [[ -e $UPLOADED_FILELIST ]]; then
    print "$UPLOADED_FILELIST already exists" 1>&2
    exit 1
fi
if [[ -e $REMOVABLE_FILELIST ]]; then
    print "$REMOVABLE_FILELIST already exists" 1>&2
    exit 1
fi

if [[ ! $#dirs -eq 1 ]]; then
    print 'most provide precisely one backup dir' 1>&2
    exit 1
fi

# based on fileUploadsByConcurrencyFactor in cmd/bundle_upload.go
if [[ $concurrency_factor -lt 5 ]]; then
    print "concurrency_factor $concurrency_factor must be set to at least 5" 1>&2
    exit 1
fi

dir_path_param=$dirs[1]

if [[ ! -d $dir_path_param ]]; then
    print "$dir_path_param doesn't exist" 1>&2
    exit 1
fi

cd $HOME

if ! print . | migrate filelist-actions --time-before $timestamp_filter_before; then
    print "$timestamp_filter_before isn't recognized by the filelist actions script" 1>&2
    exit 1
fi

###

dir_path_abs=$(cd $dir_path_param && pwd)

if [[ ! -f $HOME/.datamon/datamon.yaml ]]; then
    datamon config create \
            --name 'ransom' \
            --email 'rwilliams@oneconcern.com'
fi

sed_param='s@\(.*\)@'"${dir_path_abs}"'\/\1@'

2>/dev/null migrate generate --out - --parallel --parent $dir_path_abs | \
    sed $sed_param > ${UPLOAD_FILELIST}

###

datamon bundle upload \
        --files ${UPLOAD_FILELIST} \
        --message "upload from datamover script" \
        --path / \
        --repo $REPO \
        --label $BUNDLE_LABEL \
        --skip-on-error \
        --concurrency-factor $concurrency_factor \
        --loglevel debug

###

datamon bundle list files \
        --repo $REPO \
        --label $BUNDLE_LABEL | \
    grep -v '^Using' | \
    sed 's/name:\(.*\), size:.*, hash:.*/\1/' \
        > ${UPLOADED_FILELIST}

migrate filelist-actions \
        --filelist ${UPLOADED_FILELIST} \
        --unlink=${unlink} \
        --time-before $timestamp_filter_before \
        --out $REMOVABLE_FILELIST
