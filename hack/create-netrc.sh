#!/bin/sh

echo "machine github.com
login ${GITHUB_USER}
password ${GITHUB_TOKEN}

machine api.github.com
login ${GITHUB_USER}
password ${GITHUB_TOKEN}
" > /root/.netrc
