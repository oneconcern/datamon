#! /bin/zsh

setopt ERR_EXIT

is_latest=false

while getopts l opt; do
    case $opt in
        (l)
            is_latest=true
            ;;
        (\?)
            print Bad option, aborting.
            exit 1
            ;;
    esac
done
(( OPTIND > 1 )) && shift $(( OPTIND - 1 ))

typeset -a version_tags
version_tags=($(git tag --points-at HEAD |grep '^v' || true))

if [[ ${#version_tags} -gt 1 ]]; then
    print 'ambiguous tags at HEAD: multiple tags begin with v' 1>&2
    print -- "$version_tags" 1>&2
    exit 1
fi

if [[ ${#version_tags} -eq 1 ]]; then
    if $is_latest; then
        if &> /dev/null docker pull \
              gcr.io/onec-co/datamon-fuse-sidecar:${version_tags[1]}; then
            release_tag='latest-official-rebuild'
        else
            release_tag='latest-official-init'
        fi
    else
        release_tag=${version_tags[1]}
    fi
else
    if $is_latest; then
        release_tag='latest-unofficial'
    else
        hash=$(git show --abbrev-commit |grep '^commit' | cut -d' ' -f 2)
        release_tag="hash-${hash}"
    fi
fi

print -- ${release_tag}
