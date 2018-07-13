# Trumpet

Trumpet is a datascience tool that helps managing data at scale.
It's main goals are to version data and models together with the results they produce. 
A secondary goal to managing the data might be that it provides a scheduler to run jobs on top of a kubernetes cluster
or with a serverless framework.

The name is a reference to [Torricelli's Trumpet](https://en.wikipedia.org/wiki/Gabriel%27s_Horn) aka [Gabriel's Horn](https://en.wikipedia.org/wiki/Gabriel%27s_Horn)

## Design

There are a few major parts to this framework.  At the core there is the data management solution, which is expanded upon 
by the pipeline execution engine. 

The general idea behind the pipeline execution is that all pipelines are always executing and always waiting for new events to arrive.
Events triggers are idempotent, this means when the content of the inputs change the graph executes all the nodes in the graph are
dependent on that result. 

A processor in the pipeline is configured with a yaml or json document.

```yaml
name: the-processor-name
branch: "develop"
runtime: "reg.onec.co/flood_ml:develop"
# when the job needs to run on with access to a gpu
gpu: true
# if the job can run on the same compute resources as other processors of the same type
can_colocate: true
# define scaling limits
concurrency:
  min: 1
  max: 5
  condition: queue_depth >= 10
# optional hints for resource requirements
resources:
  cpu:
    min: 100Mi
    max: 500Mi
  mem:
    min: 200MB
    max: 2GB
# each entry is a glob pattern to select files you want included
# the paths will be preserved
content:
  - vendor/*
  - scripts/*
  - bin/*
  - requirements.txt
  - app.py
# the command to run
command:
  - python
  - app.py
secrets:
  - name: database-creds
    path: /etc/oneconcern/database-creds
configmaps:
  - name: flood-ml-config
    path: /etc/oneconcern/config
input:
  - type: repo
    # the name of the data repository
    name: flood-nldas-data
    # a branch name, tag name or commit id, when none is specified it defaults to master
    version: develop
    # the task will only see these files in /trumpet/input/flood-nldas-data
    filter: /huc2/huc8/**/*.grib
trigger:
  - type: repo
    # the name of the data repository
    name: flood-nldas-data
    # the task will only see these files
    selector: /huc2/huc8/*
  - type: cron # when cron is specified no other triggers can be specified
    schedule: "*/5 * * * *"
```

### Usage examples

There are different use objects that can be created with trumpet.

### Repositories

You can create, delete and list repositories. 

```sh
tpt repo list
tpt repo get --name hello-there
tpt repo create --name hello-there --description 'First repo in trumpet'
tpt repo delete --name hello-there
```

You can manage branches in repositories.

```sh
tpt repo branch list --repo hello-there
tpt repo branch create --repo hello-there --name new-branch
tpt repo branch delete --repo hello-there --name new-branch
tpt repo branch checkout --repo hello-there --name new-branch
```

You can add files to a branch.

```sh
curl -OL'#' https://some.site.domain/very-large-file.zip
tpt bundle add --repo hello-there very-large-file.zip
tpt bundle seal --repo hello-there --message 'first commit'
tpt bundle checkout --repo hello-there
```

You can also tag bundles

``` sh
tpt repo tag list --repo hello-there
tpt repo tag create --repo hello-there --name v0.1.0 --message "$(cat notes/v0.1.0.md)"
tpt repo tag delete --repo hello-there --name v0.1.0
tpt repo tag checkout --repo hello-there --name v0.1.0
```

### Tunes

Tunes are the tasks that can be executed with trumpet.
You can box a 

## Data Management

The data management provides a content addressable filesystem which can import data from a variety of sources.
Initially it can import data from Local files, S3, HTTP and NFS.

Data is organized as a series of repositories, which are conceptually similar to a git repo.
A repo is a pointer to a list of bundles. Each bundle contains a filesystem subtree and some metadata to describe
the content of the bundle.

### Storage layout

The storage is laid out in a way that allows S3 to efficiently store it and is similar to the way git builds its indices.
Because we use hex hashes for the file names S3's backend can optimize the storage because it provides a decent 
distribution for [partitioning on prefixes](https://docs.aws.amazon.com/AmazonS3/latest/dev/request-rate-perf-considerations.html#workloads-with-mix-request-types).

It will look something like this:

```text
.
├── blobs
│   ├── hash-1
│   ├── hash-2
│   ├── hash-3
│   ├── hash-4
│   ├── hash-5
│   ├── hash-6
│   └── hash-7
├── bundles
│   ├── bundle-1
│   │   ├── hash-1.json
│   │   ├── hash-2.json
│   │   └── hash-3.json
│   ├── bundle-2
│   │   ├── hash-1.json
│   │   ├── hash-2.json
│   │   └── hash-3.json
│   ├── bundle-3
│   │   ├── hash-1.json
│   │   ├── hash-2.json
│   │   └── hash-3.json
│   ├── bundle-1.json
│   ├── bundle-2.json
│   └── bundle-3.json
├── models
│   ├── model-1
│   │   ├── hash-1.json
│   │   ├── hash-2.json
│   │   └── hash-3.json
│   ├── model-2
│   │   ├── hash-1.json
│   │   ├── hash-2.json
│   │   └── hash-3.json
│   ├── model-3
│   │   ├── hash-1.json
│   │   ├── hash-2.json
│   │   └── hash-3.json
│   ├── model-1.json
│   ├── model-2.json
│   └── model-3.json
└── runs
    ├── hash-1.json
    ├── hash-1.log
    ├── hash-2.json
    ├── hash-2.log
    ├── hash-3.json
    └── hash-3.log
```

In our implementation all the json files will actually be entries in a global encrypted bucket in dynamodb.
The blobs will be stored as a flat list in a dedicated multi-region replicated, encrypted S3 bucket.
The *.log files will be retrieved from the kubernetes api server or through an elasticsearch query.

### Repository

A data repository is conceptually very similar to a git repository. It has a linked list of commits (bundles).
Each commit is a [merkle tree](https://en.wikipedia.org/wiki/Merkle_tree).

Each bundle is a directory named as hex representation of the merkle root hash. At the same level there is a metadata file.
For now the metadata file is a json file so that it can be read by humans and machines alike.

The [repo-name].json file contains the metadata that describes a repo. At a very minimum it contains:

* a mapping for tags and branch names to bundle ids
* a mapping of bundle ids to known branches and tags
* a description
* a mapping of teams or users and their permissions

This is collated in a single file because it helps to keep that lookup operation a single network round trip and allows for subscriptions so local caches can be maintained.

A repository 

### Bundle

A bundle represent a single changeset. It contains the delta of the file system between the previous bundle and this one.
Initially this works as a blob store with no diffing for binary files.

The [hash].json file contains the metadata that describes a commit. At a minimum this is:

* The list of parent bundles
* An ordered mapping of file path to content hash with mode flags.
* A list of authors that contributed to the bundle
* A timestamp for when the bundle was created
* Optionally a message describing the reason for the change

Bundles are uploaded as a tar file in a single upload, this upload is resumable if the client supports it.
The bundle tar file contains each file together with their blake2b checksum.

### Files

The files are stored as content addressable blobs outside of metadata like bundles and repositories.

## Handler definition

A pipeline is never really given a name, instead pipelines are formed organically by registering more execution steps.
The steps form a DAG where 1 step can either depend on a trigger like cron or webhook, or the outputs of another step.

For a pipeline the input data and output data are represented by a number of directories.
The inputs are all read-only mounts into the container, one possible way to achieve this is by symlinking all the files on the EFS file system into a location

A pipeline is essentially a single path through our continously executing DAG.
There can't be any cyclic dependencies in this graph, one way to verify this is by running the [tarjan algorithm](https://en.wikipedia.org/wiki/Tarjan%27s_strongly_connected_components_algorithm).

### File access in handlers

To access files in handlers we make them available via a fuse filesystem, this is a customization of goofys which applies the filters defined in the manifest for a handler.

### Archiving

Once we detect files are no longer in use and have gone stale after some configurable amount of time we move these files to cold storage (glacier).
To detect files that are in use we inspect all the inputs of the runs and find all the commit ids for a given data repository that no longer occur.

### Execution

The handler can be executed as a serverless function. This allows us to be fairly efficient when functions are used frequently.
It removes the burden of building a docker container because all that is packaged are the files that are required to update the runtime appropriately.

The handler will only see the data it subscribed to through a trigger.

### Triggers

There are several ways in which a model can be triggered. Some of those can be combinations of others, these combinators 

#### A data repository or processor

A trigger from an external webhook

#### A CRON job

A trigger that produces data by running a processor on a schedule

#### A kinesis stream

A trigger from a kinesis stream, when a message is received the processor is executed

#### A Webhook subscription from an external service

A trigger by a webhook subscription, whenever the webhook is received the processor runs

##### A github repository

This is a specialization of the webhook trigger, that can subscribe to events in github.

#### A HTTP stream / SSE / websocket connection

In this case we subscribe to a remote streaming protocol
