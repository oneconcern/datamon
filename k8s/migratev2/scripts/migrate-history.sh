#! /bin/bash
#
# Migrate a sequence of bundles
#
# shellcheck disable=SC1091
source /scripts/funcs.sh

echo "INFO: history migration to datamon v2 for repo ${REPO}"
sanity

# CLI input sanitization (assuming ${REPO} does not contain any blank space)
srcBundle=""
if [[ -n ${SOURCEBUNDLE} ]] ; then
  srcBundle=${SOURCEBUNDLE}
fi

# resolve starting bundle when specified by label
if [[ -n ${SOURCELABEL} ]] ; then
  srcLabel="--label \"${SOURCELABEL}\""
  if ! res=$(eval "datamon1 bundle get --repo ${REPO} ${srcLabel}" 2>&1|tail -1|cut -d, -f1) ; then
    echo "ERROR: failed to retrieve bundle for label ${SOURCELABEL} in repo: ${REPO}"
    exit 1
  fi
  srcBundle=$(trim "$res")
  if [[ -z ${srcBundle} ]] ; then
    echo "ERROR: could not find bundle for label ${SOURCELABEL} in repo: ${REPO}"
    exit 1
  fi
fi

start=""
firstDone=""
if [[ -z ${srcBundle} ]] ; then
  start="true"
  echo "INFO: migrating the entire history of bundles for repo: ${REPO}"
else
  echo "INFO: migrating a truncated history of bundles starting at ${srcBundle} for repo: ${REPO}"
fi

export REPO DESTLABEL DESTCONTEXT

# iterate through bundles sequentially
if ! eval "datamon1 bundle list --repo ${REPO} 2>&1"|grep -v '{level:"'|cut -d, -f1|\
while read -r b ;do
  bundle=$(trim "${b}")
  if [[ -z ${start} ]] ; then
    if [[ ${bundle} == "${srcBundle}" ]] ; then
      # ok found the starting point for migration
      echo "INFO: starting migration from bundle ${bundle} in repo: ${REPO}"
      start="true"
    else
      echo "INFO: skipping bundle ${bundle} in repo: ${REPO}"
      continue
    fi
  fi
  if [[ -n ${start} ]] ; then
    # tells the unitary migration job to skip params & repo checks
    echo "INFO: migrating bundle ${bundle} for repo ${REPO}"
    SOURCEBUNDLE=${bundle} SOURCELABEL="" /bin/bash /scripts/migrate.sh "${firstDone}" --sane 2>&1
    res=$?
    firstDone="--done"
    if [[ ${res} == 2 ]] ; then
      echo "WARNING: bundle ${bundle} is already existing: skipping"
      continue
    fi
    if [[ ${res} != 0 ]] ; then
      echo "ERROR: interrupted migration, ${bundle} was not fully migrated"
      exit 1
    fi
  fi
done ; then
  echo "ERROR: interrupted migration"
  exit 1
fi
echo "INFO: done with history migration of repo: ${REPO}"
