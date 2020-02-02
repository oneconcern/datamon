# Migrate to v2

## What does this do?
This k8 chart provides a job to migrate data from a datamon v1 repo to datamon v2.

* BundleIDs (ksuid) are preserved by the migration. This means you MUST migrate your data to a different blob bucket than the original one.

## What this doesn't do

* v1 buckets, metadata and blobs are not altered or destroyed by the migration. A manual cleanup of the gcs resources has to be carried out after the migration has been successfully tested
* the migrated repo and bundle is signed by the user carrying out the migration. The track of the original contributor is lost.
* kubernetes resources are not automatically relinquished upon completion: this is on purpose to allow for log inspection. k8s resources are dropped using `helm remove`

## Features

Existing repo description, bundle commit message and labels are migrated.
A new additional label may be specified for the migrated bundle.

### Migrating a single bundle from a repo

This is the default operating mode.

By default, only the most recent bundle ("commit") is migrated.
Optional parameters allow to pick a specific bundleID or label to pick for migration.

### Migrating all or several bundles from a repo

This operating mode is specified with `--set history=true`.

In this mode, all bundles are migrated sequentially in their order of creation.

**NOTE**: the staging storage must be able to hold the largest bundle in the the repo (not all bundles).

## Prerequisites

* `helm` (this has been tested with helm v2.16.1)
* `helm tiller` plugin (`helm plugin install https://github.com/rimusz/helm-tiller`)
* logged on `gcloud`, with proper scopes (see workshop runbook)
* you should know the size of your bundle on disk (by default a 10Gi staging volume is allocated)

> **NOTE**: I picked helm to drop resources more easily after completion (configmaps, secret, PVC, job...)
> and to have a convenient CLI to parameterize the job.
>
> Besides, I found the unique release naming convenient to avoid duplicate jobs.
> If you don't like helm, you may bypass the helm deployment and use the tool as a mere template renderer (see below).

## Options

Optional parameters allow to truncate the migrated history, starting from a
specific bundleID or label.

The job accepts optional inputs:

* `--set-file config1=~/.datamon/datamon.yaml`: the datamon v1 config  - defaults to gs://datamon-meta-data and gs://datamon-blob-data
* `--set-file config2=~/.datamon2/datamon.yaml` the datamon v2 config - defaults to gs://workshop-config bucket)

* `--set stagingSize={disk size}}`: the size of the storage allocated as staging for downloading bundles
* `--set newLabel={label}`: a new label for the migrated bundle (defaults to none); history mode: the label is set on the last migrated bundle
* `--set label={label}`: single mode: the label of the bundle to migrate (defaults to none); history mode: the label of the bundle from which to start migration (defaults to none)
* `--set bundle={ksuid}`: single mode : the ID of the bundle to migrate (defaults to latest) ; history mode: the ID of the bundle from which to start migration (defaults to first)
* `--set context={context}`: the datamon v2 context to use (default is set by the config2, or if none specified, "dev")

**NOTE**: label specification overturns the bundleID specification

### Single bundle mode

Extra options:
* bundleID or label to download and migrate (defaults to latest)

### Multiple bundles mode

Specify the mode on the `helm` command line: `--set history=true`

Extra options:
* bundleID or label from which to start the migration (defaults to first bundle created)

## Usage for common use cases

### Full migration of a single repo
```
helm tiller run helm install \
--name migrate-my-repo \
-f values.default.yaml \
--set history=true
--set repo=my-repo \
--set newLabel="new datamon V2 baseline" \
--set stagingSize=100Gi \
--set-file secret=~/.config/gcloud/application_default_credentials.json \
.
```

### Truncated migration of a single repo

Migrate all bundles starting the new repo at label "my-label".

```
helm tiller run helm install \
--name migrate-my-repo \
-f values.default.yaml \
--set history=true
--set label="my-label" \
--set repo=my-repo \
--set newLabel="new datamon V2 baseline" \
--set stagingSize=100Gi \
--set-file secret=~/.config/gcloud/application_default_credentials.json \
.
```

### Migrating several repos in one go

A job can only migrate one repo. You can script helm deploys to launch several jobs.
In that case, you might want to deploy your credentials only once for all jobs.

You may use `scripts/migrate-repos.sh` locally to do just that.

This script will probably need some adjustment regarding the required size of the staging volume, reuse of your
own datamon configs, etc.

Usage:
```
./scripts/migrate-repos.sh {repo-1}[{repo-2}...]

kubectl logs   -l 'app.kubernetes.io/name=migratev2' --tail -1|grep "INFO: done with history"

./scripts/migrate-repos.sh --done {repo-1}[{repo-2}...]
```

## Usage

The job requires the following inputs:
- the repo name
- your cloud credentials passed as a file

Deploy the job with args the `helm` command line:
```
helm tiller run helm install \
--name migrate-my-repo \
-f values.default.yaml \
--set repo=my-repo \
--set newLabel="new datamon V2 baseline" \
--set stagingSize=100Gi \
--set-file secret=~/.config/gcloud/application_default_credentials.json \
.
```

You can then monitor the logs of the migration pod:
```
kubectl logs -f -l job-name=migrate-my-repo --tail -1
```

To capture the logs of all migration pods:
```
kubectl logs  -l app.kubernetes.io/name=migratev2 --tail -1
```

**NOTE(1)**: you might want to manage the secret yourself and set the name of this secret instead of the one created by the chart: `--set secretName={my secret}`.
The key is expected to be the file name of the credentials file, e.g. `application_default_credentials.json`.

**NOTE(2)**: since individual jobs are considered as "releases", we advise to use the repo name in the release name (e.g. "migrate-my-repo").

## Reusing local datamon configs

**IMPORTANT**: if you have a "credential" key in your config file, please remove it as it will conflict with the container's own location.

```
helm tiller run helm install \
--name migrate-my-repo \
-f values.default.yaml \
--set repo=my-repo \
--set newLabel="new datamon V2 baseline" \
--set stagingSize=100Gi
--set-file config1=~/datamon/datamon.yaml \
--set-file config2=~/datamon2/datamon.yaml \
--set-file secret=~/.config/gcloud/application_default_credentials.json \
.
```

### Other tuning parameters

You may specify those on the command line or add additional yaml configs to feed `helm`:

* `--namespace {my-namespace}`
* `--set resources={kubernetes resources spec}`: specify the resources for the pod (CPU, RAM)
* `--set backoffs=n`: allows for n restarts of failed jobs (defaults to none)
* `--set ttl=n`: kubernetes will wipe out resources n seconds after the job has completed (defaults to 86400s or 1 day)
* `--set toleration={kubernetes toleration spec}`: specify the toleration for the pod
* `--set affinity={kubernetes afinity spec}`: specify the cluster affinity for the pod

## Cleaning resources

We want pods to remain available to inspect logs etc.

Deletion of the kubernetes resources is carried out explicitly with:
```
helm tiller run -- helm delete --purge migrate-my-repo
```
A TTL period is set in kubernetes for the pod to expire after completion (24h by default).

**NOTE**: TTL will delete the pod and unbind the persistent volume claim, which in turn releases the persistent volume.
However, the job and the config objects will only be removed by the `helm delete` command.

**NOTE**: it is most important to clean the secret which is accessible to others in the cluster. This is carried out by `helm delete`.

**NOTE**: it is important to relinquish the staging storage after usage, which is a billed resource. This is carried out by `helm delete`.

## I have issues with helm, how do I work around these?

The chart is using helm hooks in order to control all needed resources and possibly delete them after the job has completed (automatic deletion is disabled for now).
Depending on your version of helm, you may stumble on some issues, so you may want to deploy the job without helm.

1. Resolve the chart template
```
helm template migrate-my-repo -f values.default.yaml --set repo=my-repo {...} .  > migrate-my-repo.yaml
```

2. Deploy job resources manually with `kubectl`
```
kubectl apply -f migrate-my-repo
```

3. Remove resources manually: job, configmaps, secret, persistent volume claim

# A job crashed. How do I recover?

For some reason (hopefully not a bug in the migration chart...), the migration process may have been interrupted.

You may safely restart a new migration job with the same parameters.

Since we are preserving bundleIDs when migrating, bundles which have already been migrated
will be skipped when resuming the migration.
