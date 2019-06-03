[![CircleCI](https://circleci.com/gh/oneconcern/datamon/tree/master.svg?style=svg&circle-token=e827ee1509892d8ba85db2a819b692ca451a7a97)](https://circleci.com/gh/oneconcern/datamon/tree/master)

# CLI Guide

Make sure your gcloud credentials have been setup.
```$bash
gcloud auth application-default login
```
Download the datamon binary for mac or for linux on the [Releases Page](https://github.com/oneconcern/datamon/releases/)
Example:
```$bash
tar -zxvf datamon.mac.tgz
```

##### Configure datamon

For non kubernetes use, it's necessary to supply gcloud credentials.

```bash
# Replace path to gcloud credential file. Use absolute path
% datamon config create --email ritesh@oneconcern.com --name "Ritesh H Shukla" --credential /Users/ritesh/.config/gcloud/application_default_credentials.json
```

Inside a kubernetes pod, Datamon will use kubernetes service credentials.
```bash
% datamon config create --name "Ritesh Shukla" --email ritesh@oneconcern.com
```

Check the config file, credential file will not be set in kubernetes deployment.
```bash
% cat ~/.datamon/datamon.yaml
metadata: datamon-meta-data
blob: datamon-blob-data
email: ritesh@oneconcern.com
name: Ritesh H Shukla
credential: /Users/ritesh/.config/gcloud/application_default_credentials.json
```

##### Create repo
Datamon repos are analogous to git repos.

```bash
% datamon repo create  --description "Ritesh's repo for testing" --repo ritesh-datamon-test-repo
```

##### Upload a bundle.
The last line prints the commit hash.
If the optional `--label` is omitted, the commit hash will be needed to download the bundle.
```bash
% datamon bundle upload --path /path/to/data/folder --message "The initial commit for the repo" --repo ritesh-test-repo --label init
Uploaded bundle id:1INzQ5TV4vAAfU2PbRFgPfnzEwR
```

##### List bundles
List all the bundles in a particular repo.
```bash
% datamon bundle list --repo ritesh-test-repo
Using config file: /Users/ritesh/.datamon/datamon.yaml
1INzQ5TV4vAAfU2PbRFgPfnzEwR , 2019-03-12 22:10:24.159704 -0700 PDT , Updating test bundle
```

##### List labels
List all the labels in a particular repo.
```bash
% datamon label list --repo ritesh-test-repo
Using config file: /Users/ritesh/.datamon/datamon.yaml
init , 1INzQ5TV4vAAfU2PbRFgPfnzEwR , 2019-03-12 22:10:24.159704 -0700 PDT
```

##### Download a bundle

Download a bundle by either hash

```bash
datamon bundle download --repo ritesh-test-repo --destination /path/to/folder/to/download --bundle 1INzQ5TV4vAAfU2PbRFgPfnzEwR
```

or label

```bash
datamon bundle download --repo ritesh-test-repo --destination /path/to/folder/to/download --label init
```

##### List bundle contents
List all files in a bundle
```bash
datamon bundle list files --repo ritesh-test-repo --bundle 1ISwIzeAR6m3aOVltAsj1kfQaml
```

Also uses `--label` flag as an alternate way to specify the bundle in question.

##### Download a file
Download a single file from a bundle
```bash
datamon bundle download file --file datamon/cmd/repo_list.go --repo ritesh-test-repo --bundle 1ISwIzeAR6m3aOVltAsj1kfQaml --destination /tmp
```

Can also use the `--label` as an alternate way to specify the particular bundle.

##### Set a label

```bash
% datamon label set --repo ritesh-test-repo --label anotherlabel --bundle 1ISwIzeAR6m3aOVltAsj1kfQaml
Uploaded bundle id:1INzQ5TV4vAAfU2PbRFgPfnzEwR
```

Labels are a mapping type from human-readable strings to commit hashes.

There's one such map per repo, so in particular setting a label or uploading a bundle
with a label that already exists overwrites the commit hash previously associated with the
label:  There can be at most one commit hash associated with a label.  Conversely,
multiple labels can refer to the same bundle via its commit hash.

# Feature requests and bugs

Please file GitHub issues for features desired in addition to any bugs encountered.

# Datamon

Datamon is a datascience tool that helps managing data at scale.
The primary goal of datamon is to allow versioned data creation, access and tracking
in an environment where data repositories and their lifecycles are linked.

Datamon links the various sources of data, how they are processed and tracks the
output/new data that is generated from the existing data.

## Design

Datamon is composed of
1. Datamon Core
   1. Datamon Content Addressable Storage
   2. Datamon Metadata
2. Data access layer
   1. CLI
   2. FUSE
   3. SDK based tools
3. Data consumption integrations.
   1. CLI
   2. Kubernetes integration
   3. InPod Filesystem
   3. GIT LFS
   4. Jupyter notebook
   4. JWT integration
4. ML/AI pipeline run metadata: Captures the end to end metadata for a ML/AI pipeline runs.
5. Datamon Query: Allows introspection on pipeline runs and data repos.

![Architecture Overview](/docs/ArchitectureOverview.png)

## Data Storage

Datamon includes a
1. Blob storage: Deduplicated storage layer for raw data
2. Metadata storage: A metadata storage and query layer
3. External storage: Plugable storage sources that are referenced in bundles.

For blob and metadata storage datamon guarantees geo redundant replication of data and is able to withstand
region level failures.

For external storage based on the external source, the redundancy and ability to access can vary.

### Data modeling

***Repo***: Analogous to git repos. A repo in datamon is a dataset that has a unified lifecycle.
***Bundle***: A bundle is a point in time readonly view of a rep:branch and is composed of individual files. Analogous to a commit in git.
***Labels***: A name given to a bundle, analogous to tags in git. Example: Latest, production

Planned features:

***Branch***: A branch represents the various lifecycles data might undergo within a repo.
***Runs***: ML pipeline run metadata that includes the versions of compute and data in use for a given run of a pipeline.

![Datamon Model](/docs/DatamonModel.png)

## Data Access layer

Data access layer is implemented in 3 form factors
1. CLI Datamon can be used as a standalone CLI provided developer has access privileges to the backend
storage. A developer can always setup datamon to host their own private instance for managing and
tracking their own data.
2. Filesystem: A bundle can be mounted as a file system in Linux or Mac and new bundles can be generated as well.
3. Specialized tooling can be written for specific use cases. Example: Parallel ingest into a bundle for high scaled out throughput.

## Data consumption integration

### Kubernetes integration

Datamon integrates with kubernetes to allow for pod access to data and pod execution synchronization based on dependency on data.
Datamon also caches data within the cluster and informs the placement of pods based on cache locality.

### GIT LFS

Datamon will act as a backend for [GIT LFS](https://github.com/oneconcern/datamon/issues/79

### Jupyter notebook.

[Datamon allows for Jupyter notebook](https://github.com/oneconcern/datamon/issues/80) to read in bundles in a repo and process them and create new
bundles based on data generated

### Data access layer
Datamon API/Tooling can be used to write custom services to ingest large data sets into datamon.
These services can be deployed in kubernetes to manage the long duration ingest.

This was used to move data from AWS to GCP.
