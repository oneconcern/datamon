# Design

## Components

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

![Datamon Model](/docs/DatamonModel.png)

## Data Access layer

Data access layer is implemented in 3 form factors
1. CLI Datamon can be used as a standalone CLI provided developer has access privileges to the backend
storage. A developer can always setup datamon to host their own private instance for managing and
tracking their own data.
2. Filesystem: A bundle can be mounted as a file system in Linux or Mac and new bundles can be generated as well.
3. Specialized tooling can be written for specific use cases. Example: Parallel ingest into a bundle for high scaled out throughput.

## Data consumption integration

### GIT LFS

Datamon will act as a backend for [GIT LFS](https://github.com/oneconcern/datamon/issues/79)

### Jupyter notebook.

[Datamon allows for Jupyter notebook](https://github.com/oneconcern/datamon/issues/80) to read in bundles in a repo and process them and create new
bundles based on data generated

### Data access layer

Datamon API/Tooling can be used to write custom services to ingest large data sets into datamon.
These services can be deployed in kubernetes to manage the long duration ingest.

