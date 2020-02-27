#! /bin/bash
# Usage:
# Start jobs:  migrate-repos.sh {repo...}
# Delete jobs: migrate-repos.sh --done {repo...}
#
if [[  ${1:=""} == "--done" ]] ; then
  shift
  for repo in "$@" ; do
    jobRelease="mig-datamon-${repo}"
    helm tiller run -- helm delete --purge "${jobRelease}" .
  done
  exit 0
fi
for repo in "$@" ; do
  jobRelease="mig-datamon-${repo}"
  stagingSize="50Gi"
  helm tiller run -- helm install \
    -n "${jobRelease}" \
    -f values.default.yaml \
    --set stagingSize=${stagingSize} \
    --set-file secret=~/.config/gcloud/application_default_credentials.json \
    --set repo="${repo}" \
    --set newLabel="${repo} migrated to v2" \
    --set-file config1=~/.datamon/datamon.yaml \
    --set-file config2=~/.datamon2/datamon.yaml \
    --set history=true \
    .
done
