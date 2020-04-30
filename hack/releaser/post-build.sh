#! /bin/bash
#
# Post build script for goreleaser
#
# * This runs upx to compress binaries
#
# NOTE: unfortunately, we are compelled to do that because
# goreleaser post hook does not run in the build context
# to resolve templated attributes.
# See github.com/goreleaser/goreleaser/issues/1261
find dist -type f -name "$1" -exec xargs upx {} ';'
