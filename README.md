# CLI Guide

Make sure your gcloud credentials have been setup.
```$bash
gcloud auth application-default login
```
Download the datamon binary for mac or for linux on the [Releases Page](https://github.com/oneconcern/datamon/releases/tag/0.1)
Example:
```$bash
tar -zxvf datamon.mac.tgz 
```

Configure datamon (for non kubernetes use)

Example:
```$bash
# Replace path to gcloud credential file. Use absolute path
./datamon config create --email ritesh@oneconcern.com --name "Ritesh H Shukla" --credential /Users/ritesh/.config/gcloud/application_default_credentials.json
```

Configure datamon inside a pod. Datamon will use kubernetes service credentials.
```bash
~/datamon config create --name "Ritesh Shukla" --email ritesh@oneconcern.com
```

Check the config file, credential file will not be set in kubernetes deployment.
```bash
# cat ~/.datamon/datamon.yaml 
metadata: datamon-meta-data
blob: datamon-blob-data
email: ritesh@oneconcern.com
name: Ritesh H Shukla
credential: /Users/ritesh/.config/gcloud/application_default_credentials.json
```

Create repo analogous to git repo
```bash
./datamon repo create  --description "Ritesh's repo for testing" --repo ritesh-datamon-test-repo  
```

Upload a bundle, the last line prints the commit hash. This will be needed for downloading the bundle
```bash
#./datamon bundle upload --folder /path/to/data/folder --message "The initial commit for the repo" --repo ritesh-test-repo
..
..
Uploaded bundle id:1INzQ5TV4vAAfU2PbRFgPfnzEwR 
```

List bundles in a repo
```bash
#./datamon bundle list --repo ritesh-test-repo                                                                                                                
Using config file: /Users/ritesh/.datamon/datamon.yaml
1INzQ5TV4vAAfU2PbRFgPfnzEwR , 2019-03-12 22:10:24.159704 -0700 PDT , Updating test bundle
```

Download a bundle
```bash
./datamon bundle download --repo ritesh-test-repo --destination /path/to/folder/to/download --bundle 1INzQ5TV4vAAfU2PbRFgPfnzEwR
```

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
1. Data Storage
2. Data access layer
   1. CLI
   2. FUSE
   3. SDK based tools
3. Data consumption integrations.
   1. CLI
   2. Kubernetes integration
   3. GIT LFS
   4. Jupyter notebook
   4. JWT integration

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

***Branch***: A branch represents the various lifecycles data might undergo within a repo.

***Bundle***: A bundle is a point in time readonly view of a rep:branch and is composed of individual files. Analogous to a commit in git.

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

### JWT integration

Datamon can serve bundles as well as consume data that is authenticated via JWT

# Getting started guide.
# Kubernetes Guide
# GIT
