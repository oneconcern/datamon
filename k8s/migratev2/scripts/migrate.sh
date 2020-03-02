#! /bin/bash
#
# Migrate one bundle at at time
#
# shellcheck disable=SC1091
source /scripts/funcs.sh

# check if launched from a sequence: perform checks and repo creation only once
done=""
sane=""
for arg in "$@" ; do
  case ${arg} in
  --done)
    done="true"
    ;;
  --sane)
   sane="true"
    ;;
 esac
done

if [[ -z ${done} && -z ${sane} ]] ; then
  echo "INFO: retrieving repo to migrate, with description: ${REPO}"
  sanity
else
  echo "INFO: sanity checks already done. Skipped"
fi

# CLI input sanitization (assuming ${REPO} does not contain any blank space)
srcBundle=""
if [[ ! -z ${SOURCEBUNDLE} ]] ; then
  srcBundle="--bundle \"${SOURCEBUNDLE}\""
fi
if [[ ! -z ${SOURCELABEL} ]] ; then
  srcBundle="--label \"${SOURCELABEL}\""
fi
targetContext=""
if [[ ! -z ${DESTCONTEXT} ]] ; then
  targetContext="--context \"${DESTCONTEXT}\""
fi

if [[ -z ${done} ]] ; then
  if ! res=$(datamon1 repo get --repo "${REPO}" 2>&1|grep -v '{level="'|cut -d, -f2) ; then
    echo "ERROR: Failed to get repo ${REPO} on V1 config: ${res}"
    exit 1
  fi
  # retrieve the original description of the repo
  description=$(trim "$res")
  if [[ -z ${description} ]] ; then
    description="Migrated repo ${REPO}"
  fi
  echo "INFO: description for ${REPO}: ${description}"
fi

echo "INFO: retrieving bundle to migrate"
if ! res=$(eval "datamon1 bundle get --repo \"${REPO}\" $srcBundle" 2>&1|grep -v '{"level"'|grep -v '^Using') ; then
  echo "ERROR: Failed to get bundle $srcBundle in repo ${REPO} on V1 config: ${res}"
  exit 1
fi
if [[ ${res} =~ "didn't find" ]] ; then # not sure about the exit code from datamon v1 here - best to check twice
  echo "ERROR: Failed to get bundle $srcBundle in repo ${REPO} on V1 config: ${res}"
  exit 1
fi

# retrieve the original message of the bundle
mres=$(echo "${res}"|cut -d, -f3)
message=$(trim "${mres}")
if [[ -z ${message} ]] ; then
  message="Migration to datamon V2"
fi
echo "INFO: bundle commit message: ${message}"

# retrieve the original bundleID
bres=$(echo "${res}"|cut -d, -f1)
bundleID=$(trim "${bres}")

if [[ -z ${bundleID} ]] ; then
  echo "ERROR: Could not extract bundleID from ${res}"
  exit 1
fi
echo "INFO: original bundleID: ${bundleID}"

# retrieve existing labels for this bundle
function setLabels {
  echo "INFO: retrieving existing labels for the bundle to migrate on repo: ${REPO}"
  datamon1 label list --repo "${REPO}" 2>&1|grep -v '{level:"'|grep "${bundleID}"|cut -d, -f1|\
  while read -r l ; do
    label=$(trim "${l}")
    if [[ ! -z ${label} && ${label} != "${DESTLABEL}" ]] ; then
      echo "INFO: adding label \"${label}\" to new bundle ID: ${REPO}"
      if ! eval "datamon2 label set --repo \"${REPO}\" --bundle \"${bundleID}\" --label \"${label}\" $targetContext" ; then
        echo "ERROR: Failed to migrate existing labels into V2 repo ${REPO} for V1 bundle ${bundleID}"
      fi
    fi
  done
  if [[ ! -z ${DESTLABEL} ]] ; then
    echo "INFO: setting additional label ${DESTLABEL} for repo: ${REPO}"
    if ! eval "datamon2 label set --repo \"${REPO}\" --bundle \"${bundleID}\" --label \"${DESTLABEL}\" $targetContext" ; then
      echo "ERROR: Failed to apply new label o V2 repo ${REPO} for bundle ${bundleID}"
    fi
  fi
}

dest=${STAGING}/${bundleID}
rm -rf "${dest}"
if ! mkdir -p "${dest}" ; then
  echo "ERROR: Cannot write to staging ${STAGING}"
  exit 1
fi

if [[ -z ${done} ]] ; then
  echo "INFO: checking if repo already exists in V2 store: ${REPO}"
  if ! eval "datamon2 repo get --repo \"${REPO}\" $targetContext" ; then
    echo "INFO: Repo ${REPO} does not exist in the V2 config yet. Creating it"
    if ! eval "datamon2 repo create --repo \"${REPO}\" --description \"${description}\" $targetContext" ; then
      echo "ERROR: Failed to create V2 repo ${REPO}"
      exit 1
    fi
  else
    echo "INFO: Repo ${REPO} already exists in the V2 config. Proceeding with copying the bundle"
  fi
fi

 echo "INFO: checking for already existing bundle ${bundleID} in target store for repo: ${REPO}"
 if ! eval "datamon2 bundle get --repo \"${REPO}\" --bundle \"${bundleID}\"" ; then
   echo "WARNING: bundle ${bundleID} is already existing in the store with target repo: ${REPO}"
   echo "ensuring labels in target bundle ${bundleID}: ${REPO}"
   setLabels
   # when looping through bundles, caller migth want to continue
   exit 2
fi

echo "INFO: downloading bundle ${bundleID}: ${REPO}"
if ! eval "datamon1 bundle download --repo \"${REPO}\" --bundle \"${bundleID}\" srcBundle --destination ${dest}" ; then
  echo "ERROR: Download failed for ${bundleID} in repo: ${REPO}"
  exit 1
fi

echo "INFO: uploading bundle, with bundleID preservation ${bundleID}: ${REPO}"
if ! eval "datamon2 bundle upload --repo \"${REPO}\" --path \"${dest}\" --bundle \"${bundleID}\" --message \"${message}\" $targetContext" ; then
  echo "ERROR: Failed to upload bundle ${bundleID} into V2 repo ${REPO}"
  exit 1
fi

setLabels

echo "INFO: Bundle ${bundleID} successfully migrated for repo ${REPO}"
rm -rf "${dest}"
exit 0
