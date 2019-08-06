[![CircleCI](https://circleci.com/gh/oneconcern/datamon/tree/master.svg?style=svg&circle-token=e827ee1509892d8ba85db2a819b692ca451a7a97)](https://circleci.com/gh/oneconcern/datamon/tree/master)

# CLI Guide

Make sure your gcloud credentials have been setup.
```$bash
gcloud auth application-default login
```
Download the datamon binary for mac or for linux on the
[Releases Page](https://github.com/oneconcern/datamon/releases/)
or use the
[shell wrapper](#os-x-install-guide)

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

# Kubernetes sidecar guide

Current use of Datamon at One Concern with respect to intra-Argo workflow usage relies on the
[kubernetes sidecar](https://kubernetes.io/docs/tasks/access-application-cluster/communicate-containers-same-pod-shared-volume/)
pattern where a shared volume is used as the transport layer for application layer
communication to coordinate between the _main container_, where a data-science program
accesses data provided by Datamon and produces data for Datamon to upload, and the
_sidecar container_, where Datamon provides data for access (via streaming through
main memory directly from GCS) and then, after the main container is done outputting
data to a shared Kubernetes volume, uploads the results of the data-science program
to GCS.  Ensuring that, for example, the streaming data is ready for access (sidecar to
main-container messaging) as well as notification that the data-science program has
produced output data to upload (main-container to sidecar messaging), is the responsibility
of a couple of shell scripts that both ship inside the `gcr.io/onec-co/datamon-fuse-sidecar`
container, which is versioned along with
[github releases](https://github.com/oneconcern/datamon/releases/)
of the desktop binary:  to access release `0.4` as listed on the github releases page,
use the tag `v0.4` as in `gcr.io/onec-co/datamon-fuse-sidecar:v0.4` when
writing Dockerfiles or Kubernetes-like YAML that accesses the sidecar container image.

Users need only place the `wrap_application.sh` script located in the root directory
of the sidecar container within the main container.  This can be accomplished via
an `initContainer` without duplicating version of the Datamon sidecar image in
both the main application Dockerfile as well as the YAML.  When using a block-storage GCS
product, we might've specified a data-science application's Argo DAG node with something
like

```yaml
command: ["app"]
args: ["param1", "param2"]
```

whereas with `wrap_application.sh` in place, this would be something to the effect of

```yaml
command: ["/path/to/wrap_application.sh"]
args: ["-c", "/path/to/coordination_directory", "--", "app", "param1", "param2"]
```

That is, `wrap_application.sh` has the following usage

```shell
wrap_application.sh -c <coordination_directory> -- <application_command>
```

where `<coordination_directory>` is an empty directory in a shared volume
(an
[`emptyDir`](https://kubernetes.io/docs/concepts/storage/volumes/#emptydir)
using memory-backed storage suffices).  In the case of Argo workflows in particular,
the empty directory (and not necessarily the volume) ought to be specific to a
particular DAG node (i.e. Kubernetes pod).  Each node uses a unique directory.
Meanwhile, `<application_command>` is the data-science application command exactly as it
would appear without the wrapper script.
That is, the wrapper script, relies the
[conventional UNIX syntax](http://zsh.sourceforge.net/Guide/zshguide02.html#l11)
for stating that options to a command are done being declared.

Meanwhile, `wrap_datamon.sh` similarly accepts a single `-c` option to specify the
location of the coordination directory.
Additionally, `wrap_datamon.sh` accepts a `-d` option.  The parameters to this option are
among the standard Datamon CLI commands:

* `config`
* `bundle mount`
* `bundle upload`

Multiple (or none) `bundle mount` and `bundle upload` commands may be specified,
and at most one `config` command is allowed so that an example `wrap_datamon.sh`
YAML might be

```yaml
command: ["./wrap_datamon.sh"]
args: ["-c", "/tmp/coord", "-d", "config create --name \"Coord\" --email coord-bot@oneconcern.com", "-d", "bundle upload --path /tmp/upload --message \"result of container coordination demo\" --repo ransom-datamon-test-repo --label coordemo", "-d", "bundle mount --repo ransom-datamon-test-repo --label testlabel --destination /tmp --mount /tmp/mount --stream"]
```

or from the shell

```shell
./wrap_datamon.sh -c /tmp/coord -d 'config create --name "Coord" --email coord-bot@oneconcern.com' -d 'bundle upload --path /tmp/upload --message "result of container coordination demo" --repo ransom-datamon-test-repo --label coordemo' -d 'bundle mount --repo ransom-datamon-test-repo --label testlabel --destination /tmp --mount /tmp/mount --stream'
```

where, in particular, the `-d` (Datamon) options passed to the shell wrapper are
scalars.

# OS X install guide

The recommended way to install datamon in your local environemnt is to use the
`deploy/datamon.sh` wrapper script.  This script is responsible for downloading
the datamon binary from the [Releases Page](https://github.com/oneconcern/datamon/releases/),
keeping a local cache of binaries, and `exec`ing the binary.  So parameterization
of the shell script is the same as parameterization as the binary:  the shell script
is transparent.

Download the script, set it to be executable, and then try the `version` verb in the
wrapped binary to verify that the binary is installed locally.  There are several
auxilliary programs required by the shell script such as `grep` and `wget`.  If these
are not installed, the script will exit with a descriptive message, and the missing
utility can be installed via [`brew`](https://docs.brew.sh/) or otherwise.

```
curl https://raw.githubusercontent.com/oneconcern/datamon/master/deploy/datamon.sh -o datamon
chmod +x datamon
./datamon version
```

It's probably most convenient to have the wrapper script placed somewhere on your
shell's path, of course.

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
