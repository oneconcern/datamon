#! /bin/zsh

setopt ERR_EXIT
setopt PIPE_FAIL

perhaps_module_dirs=(pkg/)

for perhaps_module_dir in $perhaps_module_dirs; do
    find $perhaps_module_dir -type d |while read pkg_dir; do
        if find $pkg_dir -maxdepth 1 \
                |grep -q '_test.go$'; then
            print -- "building $pkg_dir test binary"
            (cd $pkg_dir \
                 && go test -c)
        fi
    done
done

# ./deploy ???
perhaps_main_root_dirs=(cmd/ internal/ hack/ pkg/)

typeset -a mains

for perhaps_main_root_dir in $perhaps_main_root_dirs; do
    find $perhaps_main_root_dir -type d \
        |while read perhaps_main_dir; do
        (cd $perhaps_main_dir \
             && find $ -type f \
                 |while read perhaps_main; do
                 if cat $perhaps_main \
                         |grep -q '^package\s*main'; then
                     mains=($mains $perhaps_main)
                 fi
                 if [[ $#mains -eq 1 ]]; then
                     print -- "attempting $$perhaps_main build"
                     go build
                 else
                     if [[ $#mains -eq 0 ]]; then continue; fi
                     print -- "unsure how to build multiple mains" \
                           "in same directory $perhaps_main_dir build"
                 fi
             done)
    done
done
