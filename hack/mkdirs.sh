#!/bin/bash

rootdir=/tmp/datamondata

rm -rf "$rootdir"
mkdir -p "$rootdir"
mkdir -p "$rootdir/"{blobs,bundles,processors,runs}
mkdir -p "$rootdir/bundles/bundle-"{1..3}
mkdir -p "$rootdir/processors/processor-"{1..3}
touch "$rootdir/bundles/bundle-"{1..3}".json"
touch "$rootdir/bundles/bundle-"{1..3}"/hash-"{1..3}".json"
touch "$rootdir/processors/processor-"{1..3}".json"
touch "$rootdir/processors/processor-"{1..3}"/hash-"{1..3}".json"
touch "$rootdir/runs/hash-"{1..3}"."{json,log}
touch "$rootdir/blobs/hash-"{1..7}

tree --dirsfirst "$rootdir"
