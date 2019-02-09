# Datamon

Datamon is a data versioning, lifecycle tracking and data access platform. The primary goal of datamon is to allow version data creation, access and tracking 
in an environment where data repositories and their lifecycles are linked. An example of this linking of lifecycles is a machine learning pipeline`where
pods can consume data from other pods and generate data that other pods will consume. 

Datamon allows data science tools to manage data at scale in an autonomous manner.

## Design

Datamon is composed of 
1. Data at rest management
2. Data ingest layer
   1. CLI
   2. FUSE
   3. Parallel large scale ingest
3. Access management to data
   1. CLI
   2. Kubernetes integration
   3. GIT LFS
   4. Jupyter notebook
   4. JWT integration

## Kubernetes integration

Datamon integrates with kubernetes to allow for pod access to data and pod execution synchronization based on dependency on data.
Datamon also caches data within the cluster and informs the placement of pods based on cache locality. 

## CLI

## GIT 

## Data ingest layer

## Data at rest

### Geo replication

### Consistency of writes


