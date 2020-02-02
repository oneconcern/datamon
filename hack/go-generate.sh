#! /bin/bash
set -e -o pipefail
cd "$(git rev-parse --show-toplevel)"
# some hack to bypass codegen errors when building from CI
# this has somehow to do with arcane version management of golang.org/x/tools/go...
# cf. https://github.com/matryer/moq/issues/103
mockDir="pkg/storage/mockstorage"
mkdir -p ${mockDir}
echo "package mockstorage" > ${mockDir}/store.go
moq -out pkg/storage/mockstorage/store.go -pkg mockstorage pkg/storage Store
