# Datamon

Datamon is the data and model management and monitoring engine for OneConcern.

Datamon can be used by
1. Developers for their dev sandboxes to run experiments
2. Operations for production data ingest and processing,
3. Developers and Operations for tracking the data and model provenance that impacts the health of the over all AI platform

Datamon can be used as a standalone tool with S3 as a backing store.
Datamon also tightly integrates with kubernetes to allow for defining and executing elaborate pipelines of data
processing.

Datamon has access control built in with integration with identity management.

Datamon backend can span multiple regions and will serve as the backend to make One Concern's  service is highly available.

# Immediate goals

1. Enable teams to transition their workloads to Datamon.
   1. Containers can describe the data they need and it will be made available as a read only folder in the k8s runtime
   2. Containers are given a outgoing folder where all data written will be committed to Datamon as a new version along
      with metadata of the overall execution of the container.
2. Relation between input data, container and output data is captured and be read and queried.
   1. Input data is versioned
   2. Container/Compute version is mapped to a git commit.
   3. Output data is versioned and is coupled to the Input data and Compute version.
   4. Output data can be used as an Input for a compute run.
3. Developers can pull and push data to Datamon and use it for their personal sandboxes
   1. Developers can fork a dataset and make changes
4. Collect performance metrics for pipeline
5. Measure cost for all aspects of the pipeline
   1. Ex: Measure the benefits of de-duplication

# Post Immediate goals

0. Integrate with identity management and implement access control
1. Reduce the end to end pipeline execution
   1. Example: Include a Fuse layer to reduce the time taken to make data available and improve cashing efficiency.
2. Reduce the cost of execution
   1. Measure the code on S3 side, deduplication

# Summary and design.

Datamon hosts repos that consist of data that share a common lifecycle. Each repo is a collection of branches, bundles and tags.

Bundle: A bundle is a snapshot of the data in the repo. Two bundles can be compared for files that differ.
Branch: Branch is reflects a linear sequence of bundle updates and points to the most recent bundle in that linear sequence
Tag: Tag is a named label attached to a bundle.

You can create, delete and list repositories.

```sh
datamon repo list
datamon repo get <repo name>
datamon repo create <repo name> --description 'My first repo'
datamon repo delete <repo name>
datamon sync
```

You can manage branches in repositories.

```sh
cd <repo name>
datamon branch list
datamon branch create <branch name>
datamon branch delete <branch name>
datamon branch checkout <branch name>
datamon sync
```

You can add files to a branch.

```sh
cd <repo name>
curl -OL'#' https://some.site.domain/very-large-file.zip
datamon bundle add very-large-file.zip
datamon bundle seal --message 'first commit'
datamon bundle checkout
datamon sync
```

You can also tag bundles

``` sh
cd <repo name>
datamon tag list
datamon tag create --name v0.1.0 --message "$(cat notes/v0.1.0.md)"
datamon tag delete --name v0.1.0
datamon tag checkout --name v0.1.0
datamon sync
```

## Defining a pipeline

Datamon engine understands individual events and stages that depend on it. A pipeline is constructed by defining events
and which stages are dependent on it.

A stage is
1. A set of input data defined as ```<repo:branch>``` or ```<repo:tag>```
2. A single model ```repo:branch``` or ```repo:tag```
3. A set of output data defined as ```<repo:branch>```

There is no upfront definition of a pipeline. Each stage in the pipeline is defined to be triggered on a named event.
Thus, stages can be associated with an event in an ad-hoc fashion.
Events can be one
1. A new bundle uploaded to a branch
2. A new version of a model is available
3. A new joint release of a data and model is available.

While orchestrating a pipeline care must be taken to name pipes and dependencies.

TODO: Add how an end developer describes the yaml/cmd line for form a pipeline

## S3 Layout

A single instance of Datamon sits on top of a single bucket in which the data is organized using prefixes and delimiters.

The motivation for using a single bucket is due to
1. A single entity that needs to be managed for the entire company
2. Deduplication benefits.

```text
# Test within <> brackets is user/runtime defined.
.
├── blobs
│   ├── <hash-name>
├── <repo>-bundles
│   ├── <bundle-1>
│   │   ├── bundle-files-1.json # Files json
│   │   ├── bundle-files-2.json
│   │   ├── bundle-files-3.json
│   │   └── bundle-1.json       # Descriptor json
│   ├── <bundle-2>
│   │   ├── bundle-files-1.json
│   │   ├── bundle-files-2.json
│   │   ├── bundle-files-3.json
│   │   └── bundle-2.json
│   ├── <bundle-3>
│       ├── bundle-files-1.json
│       ├── bundle-files-2.json
│       ├── bundle-files-3.json
│       └── bundle-3.json
├── <repo>-models
│   ├── <models-1>
│   │   ├── models-files-1.json
│   │   ├── models-files-2.json
│   │   ├── models-files-3.json
│   │   └── models-1.json
│   ├── <models-2>
│   │   ├── models-files-1.json
│   │   ├── models-files-2.json
│   │   ├── models-files-3.json
│   │   └── models-2.json
└── <repo>-runs
│   ├── <run-1>.json
│   ├── <run-2>.json
│   └── <run-3>.json
└── repos
    ├── <repo-1>.json
    ├── <repo-2>.json
    └── <repo-3>.json
```

### Blobs

Blobs store the raw data that is chunked named with a hash based on it's content. This is based on Blake2.
Blobs are a flat namespace across all data within Datamon and thus identical across repos and bundles will be deduplicated.
This, should allow cheap ($ cost) branching of large data sets by developers.

Post MVP we need to come up with a scheme to garbage collect unreferenced blobs.

### Bundles

Bundles contain
1. List of all files and their top level blob hash
2. Unique user assigned name
3. Unique Datamon generated hash (KSUID)
4. Computation metadata that generated it (optional)
5. Input data (< set of repo:bundle>) information that was fed to the computation that generated the bundle (optional)


#### List of files

The list of files are split across multiple sortable json files, the list of files is sorted depth first. The list is split
into json files to allow for parallel population of the bundle locally on the client side.
The list of files also captures the unix attributes and extended attributes for a file.

### Models

TODO

### Repos

Repos are a logical collection of data files that are meant to live together and be managed as a unit.

### Runs

Runs is an index of the bundles that are generated from runs of computation. Each run includes information about the input
data, the computation and the output data. Runs also include extensible attributes that can be used to populate the kubernetes
pod information and overall Datamon pipeline information.

## Order of updates

1. Blobs written
2. Bundle/Model information
  1. Files json
  2. Descriptor json (Write is complete)
3. Run (Redundant with Bundle/Model)
  1. Run metadata (subset of descriptor json information)

1. Put in S3 is atomic.
2. Blake2 will generate unique hashes for blobs. Concurrent puts to the same hash should be safe
3. Bundles/Models have unique names and their hashes are sortable (KSUID)
4. Write is complete when the description of the bundle/Model is written to S3.
5. Run index can be rebuilt if the client dies before writing the Run index.
   1. Datamon Daemon can scan for bundles written after the last run that was written and generate the  missing run indexes

## Layout of data on a node
The following is describing the kubernetes integration, for a developer using Datamon the model is similar but simpler.

Layout for a Repo:Bundle/Model is based on the filesystem view for the container consuming the data/

* Volume containing blob cache is mounted R/W into the init container within a Pod.
* Blank volume is mounted R/W that will contain the filesystem view


1. Fetch the blobs needed for a file
2. If the file is larger than a single blob concatenate the blobs and copy them in the spot in the filesystem they are expected.
3. If the file is smaller than a single blob, link the file to the blob from the blob cache volume.


### Mutable view

The output volumes will be mounted to a model's pod at pre determined paths.
At the end of a successful run, the volumes will be marked to have output data.
The Daemon pod on the node will then commit the output bundle to the respective repo:branch.

# Components of Datamon

```

                    +----------------+          +------+
                    | Datamon Service+----------+  S3  |
                    +--------+-------+          +---+--+
                             |                      |
                             |                      |
             +---------------+-------+              |
             |   +-----------------+ |              |
             |   | Datamon Core Lib| |              |
             |   +-----------------+ +--------------+
             |                       |
             |  Datamon Client Lib   |
             +-----------------------+
                        +                                   +--------------------+
          +-----+       |                           +------+|Datamon Kube Service|
          | CLI +-------+                           |       +---------+----------+
          +--+--+                                   |                 |
             ++------------------------------+      |                 |
      +------+---------------+          +----+------+-----+           |
      |Datamon Init Container|          |Datamon Daemonset|           |
      +--------+-------------+          +-----------------+           |
               |                                                      |
               +------------------------------------------------------+
```

## Kubernetes integration

To integrate Datamon with kubernetes the following pods will act in unison
0. Datamon Daemon: Generate the yaml which ties the input data to the output data specification based on the compute being executed.
Schedule is also responsible for orchestration which pods are ready to be run based on the data they depend on.
1. Init Container: This container is responsible for copying the data into a pod where the compute
is expected to be run.
2. DaemonSet: This container is responsible for the commit and push of the output of the data from a compute execution and
tie in the metadata for the input, compute and output.

# Query Datamon

Datamon supports querying the runs and the bundle metadata.
