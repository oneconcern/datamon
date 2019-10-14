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
pattern wherein a shared volume (transport layer) ramifies application layer
communication to coordinate between the _main container_, where a data-science program
accesses data provided by Datamon and produces data for Datamon to upload, and the
_sidecar container_, where Datamon provides data for access (as hierarchical filesystems,
as SQL databases, etc.).
After the main container's DAG-node-specific data-science program outputs data
(to shared Kubernetes volume, to a PostgreSQL instance in the sidecar, and so on),
the sidecar container uploads the results of the data-science program to GCS.

Ensuring that data is ready for access (sidecar to main-container messaging)
as well as notification that the data-science program has
produced output data to upload (main-container to sidecar messaging),
is the responsibility of a few shell scripts shipped as part and parcel of the
Docker images that practicably constitute sidecars.
While there's precisely one application container per Argo node,
a Kubernetes container created from an arbitrary image,
sidecars are additional containers in the same Kubernetes pod
-- or Argo DAG node, we can say, approximately synonymously --
that concert datamon-based data-ferrying setups with the application container.

_Aside_: as additional kinds of data sources and sinks are added,
we may also refer to "sidecars" as "batteries," and so on as semantic drift
of the shell scripts shears away feature creep in the application binary.

There are currently two batteries-included® images

* `gcr.io/onec-co/datamon-fuse-sidecar`
  provides hierarchical filesystem access
* `gcr.io/onec-co/datamon-pg-sidecar`
  provides PostgreSQL database access

Both are versioned along with
[github releases](https://github.com/oneconcern/datamon/releases/)
of the
[desktop binary](#os-x-install-guide).
to access recent releases listed on the github releases page,
use the git tag as the Docker image tag:
At time of writing,
[v0.7](https://github.com/oneconcern/datamon/releases/tag/v0.7)
is the latest release tag, and (with some elisions)
```yaml
spec:
  ...
  containers:
    - name: datamon-sidecar
    - image: gcr.io/onec-co/datamon-fuse-sidecar:v0.7
  ...
```
would be the corresponding Kubernetes YAML to access the sidecar container image.

_Aside_: historically, and in case it's necessary to roll back to an now-ancient
version of the sidecar image, releases were tagged in git without the `v` prefix,
and Docker tags prepended `v` to the git tag.
For instance, `0.4` is listed on the github releases page, while
the tag `v0.4` as in `gcr.io/onec-co/datamon-fuse-sidecar:v0.4` was used when writing
Dockerfiles or Kubernetes-like YAML to accesses the sidecar container image.

Users need only place the `wrap_application.sh` script located in the root directory
of each of the sidecar containers within the main container.
This
[can be accomplished](https://github.com/oneconcern/datamon/blob/master/hack/k8s/example-coord.template.yaml#L15-L24)
via an `initContainer` without duplicating version of the Datamon sidecar
image in both the main application Dockerfile as well as the YAML.
When using a block-storage GCS product, we might've specified a data-science application's
Argo DAG node with something like

```yaml
command: ["app"]
args: ["param1", "param2"]
```

whereas with `wrap_application.sh` in place, this would be something to the effect of

```yaml
command: ["/path/to/wrap_application.sh"]
args: ["-c", "/path/to/coordination_directory", "-b", "fuse", "--", "app", "param1", "param2"]
```

That is, `wrap_application.sh` has the following usage

```shell
wrap_application.sh -c <coordination_directory> -b <sidecar_kind> -- <application_command>
```

where
* `<coordination_directory>` is an empty directory in a shared volume
  (an
  [`emptyDir`](https://kubernetes.io/docs/concepts/storage/volumes/#emptydir)
  using memory-backed storage suffices).  each coordination directory (not necessarily the volume)
  corresponds to a particular DAG node (i.e. Kubernetes pod) and vice-versa.
* `<sidecar_kind>` is in correspondence with the containers specified in the YAML
  and may be among
  - `fuse`
  - `postgres`
* `<application_command>` is the data-science application command exactly as it
  would appear without the wrapper script.  That is, the wrapper script, relies the
  [conventional UNIX syntax](http://zsh.sourceforge.net/Guide/zshguide02.html#l11)
  for stating that options to a command are done being declared.

Meanwhile, each sidecar's datamon-specific batteries have their corresponding usages.

##### `gcr.io/onec-co/datamon-fuse-sidecar` -- `wrap_datamon.sh`

Provides filesystem representations (i.e. a folder) of [datamon bundles](#data-modeling).
Since bundles' filelists are serialized filesystem representations,
the `wrap_datamon.sh` interface is tightly coupled to that of the self-documenting
`datamon` binary itself.

```shell
./wrap_datamon.sh -c <coord_dir> -d <bin_cmd_I> -d <bin_cmd_J> ...
```

* `-c` the same coordination directory passed to `wrap_application.sh`
* `-d` all parameters, exactly as passed to the datamon binary, except as a
  single scalar (quoted) parameter, for one of the following commands
  - `config` sets user information associated with any bundles created by the node
  - `bundle mount` provides sources for data-science applications
  - `bundle upload` provides sinks for data-science applications

Multiple (or none) `bundle mount` and `bundle upload` commands may be specified,
and at most one `config` command is allowed so that an example `wrap_datamon.sh`
YAML might be

```yaml
command: ["./wrap_datamon.sh"]
args: ["-c", "/tmp/coord", "-d", "config create --name \"Coord\" --email coord-bot@oneconcern.com", "-d", "bundle upload --path /tmp/upload --message \"result of container coordination demo\" --repo ransom-datamon-test-repo --label coordemo", "-d", "bundle mount --repo ransom-datamon-test-repo --label testlabel --mount /tmp/mount --stream"]
```

or from the shell

```shell
./wrap_datamon.sh -c /tmp/coord -d 'config create --name "Coord" --email coord-bot@oneconcern.com' -d 'bundle upload --path /tmp/upload --message "result of container coordination demo" --repo ransom-datamon-test-repo --label coordemo' -d 'bundle mount --repo ransom-datamon-test-repo --label testlabel --mount /tmp/mount --stream'
```

##### `gcr.io/onec-co/datamon-pg-sidecar` -- `wrap_datamon_pg.sh`

Provides Postgres databases as bundles and vice versa.
Since the datamon binary does not include any Postgres-specific notions,
the UI here is more decoupled than that of `wrap_datamon.sh`.
The UI is specified via environment variables
such that `wrap_datamon.sh` is invoked without parameters.

The script looks for precisely one `dm_pg_opts` environment variable
specifying global options for the entire script and any number of
`dm_pg_db_<db_id>` variables, one per database.

----

_Aside on serialization format_

Each of these environment variables each contain a serialized dictionary
according the the following format

```
<entry_sperator><key_value_seperator><entry_1><entry_seperator><entry_2>...
```

where `<entry_sperator>` and `<key_value_seperator>` are each a single
character, anything other than a `.`, and each `<entry>` is of one of
two forms, either `<option>` or `<option><key_value_seperator><arg>`.

So for example

```
;:a;b:c
```

expresses something like a Python map

```python
{'a': True, 'b' : 'c'}
```

or shell option args

```
<argv0> -a -b c
```

----

Every database created in the sidecar corresponding to a `dm_pg_db_<db_id>`
env var is uploaded to datamon and optionally initialized by a previously
uploaded database.
The opts in the above serialization format availble to specify are

* `p` IP port used to connect to the database
* `m` message written to the database's bundle
* `l` label written to the bundle
* `r` repo containing bundle
* `sr` repo containing the source bundle
* `sl` label of the source bundle
* `sb` source bundle id


that affect the availability of the database from the application container
or the upload of the database to datamon are

Meanwhile, `dm_pg_opts` uses options

* `c` the `<coord_dir>` as in the FUSE sidecar
* `V` whether to ignore postgres version mismatch,
  either `true` or `false` (for internal use)
* `S` without an `<arg>` is causes the wrapper script to sleep instead
  of exiting, which can be useful for debug.


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


# Datamover container guide

As with the [Kubernetes sidecar guide](#kubernetes-sidecar-guide), this section covers
a particular operationalization of Datamon at One Concern wherein we use the program
along with some auxilliary programs, all parameterized via a shell script and shipped
in a Docker image, in order to periodically backup a shared block store and remove
files according to their modify time.

The docker image is called `gcr.io/onec-co/datamon-datamover` and is tagged with
versions just as the Kubernetes sidecar, `v<release_number>`, where `v0.7` is the first
tag that will apply to the Datamover.

The `datamover` image contains two shell wrappers, `backup` and `datamover`.
Both fulfill approximately the same purpose, backing up files from an NFS share
to datamon.  The main difference is that `backup` uses standard *nix utils,
while `datamover` uses an auxilliary util maintained alongside datamon.
Their respective parameters are as follows:

### `backup`

* `-d` backup directory.  required if `-f` not present.
  this is the recommended way to specify files to backup from a kubernetes job.
* `-f` backup filelist.  list of files to backup.
* `-u` unlinkable filelist.  when specified, files that can be safely deleted
  after the backup are written to this list.  when unspecified, files are deleted
  by `backup`.
* `-t` set to `true` or `false` in order to run in test mode, which at present
  does nothing more than specify the datamon repo to use.


### `datamover`

* `-d` backup directory.  required.
* `-l` bundle label.  defaults to `datamover-<timestamp>`
* `-t` timestamp filter before.  a timestamp string in system local time among several formats, including
  - `<Year>-<Month>-<Day>` as in `2006-Jan-02`
  - `<Year><Month><Day><Hour><Minute>` as in `0601021504`
  - `<Year><Month><Day><Hour><Minute><second>` as in `060102150405`
  defaults to `090725000000`
* `-f` filelist directory.  defaults to `/tmp` and is the location to write
  - `upload.list`, the files that datamon will attempt to upload as part of the backup
  - `uploaded.list`, the files that have been successfully uploaded as part of the backup
  - `removable.list`, the files that have been successfully uploaded and that have a modify time before the specified timestamp filter
* `-c` concurrency factor.  defaults to 200.  tune this down in case of the NFS being hammered by too many reads during backup.
* `-u` unlink, a boolean toggle.  whether to unlink the files in `removeable.list` as part of the `datamover` script.  defaults to off/false/not present.


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
