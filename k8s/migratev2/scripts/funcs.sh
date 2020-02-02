#! /bin/bash
sanity () {
  echo "Running datamon migration: v1 => v2"
  echo "==================================="
  echo "datamon repo: ${REPO}"
  echo "datamon source bundle option: ${SOURCEBUNDLE}"
  echo "datamon source label option: ${SOURCELABEL}"
  echo "staging data mounted on: ${STAGING}"
  df -h ${STAGING}

  echo "datamon destination bundle label: ${DESTLABEL}"
  echo "datamon destination context used: ${DESTCONTEXT}"
  echo "Configuration V1:"
  cat ${HOME}/.datamon/datamon.yaml

  echo "Configuration V2:"
  cat ${HOME}/.datamon2/datamon.yaml

  type datamon1
  type datamon2

  # Sanity checks
  if [[ -z ${GOOGLE_APPLICATION_CREDENTIALS} ]] ; then
    echo "ERROR: Credentials location is required"
    exit 1
  fi
  if [[ -z ${REPO} ]] ; then
    echo "ERROR: Repo parameter is required"
    exit 1
  fi
  if [[ -z ${STAGING} ]] ; then
    echo "ERROR: Staging location parameter is required"
    exit 1
  fi
  echo "INFO: Credentials located at: ${GOOGLE_APPLICATION_CREDENTIALS}"
  if [[ ! -f ${GOOGLE_APPLICATION_CREDENTIALS} ]] ; then
    echo "Could not find credentials file"
    exit 1
  fi
  if [[ -z ${SOURCEBUNDLE} ]] ; then
    echo "INFO: Migrating latest bundle"
  fi
  echo "==================================="
}
