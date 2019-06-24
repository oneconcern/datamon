#! /bin/sh

# this shell wrapper around the cli_fuse_tests.go to remind how to run the tests.
# these tests are not run as part of CI since FUSE is unavailable via CircleCI.

(cd .. && go build)

go test -tags fuse_cli -list Mount |grep -v '^ok' |while read -r test_name; do
    umount /tmp/mmp
    rm -r /tmp/mmp /tmp/mmfs
    go test -v -tags fuse_cli -run '^'"${test_name}"'$'
done
