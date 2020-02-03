# WAL

## Background

Datamon is primarily a client side library that implements logic for storing and retrieving data and lineage in 
varying backends based on basic object storage interface.

To keep operational model simple and introduce as few sources of failures in the core datapath, there is no Datamon
service in the data path. Also, for simplifying backup store, policy for archiving and disaster recovery planning, the 
data stored in the object store is complete and the single source of truth.

There is an additional need to be able to index and metadata for data and lineage to allow for searches. To avoid
having to rescan all the metadata to detect new updates, WAL is implemented that loosely orders the sequence in which 
updates are made in a Datamon instance.

Each Datamon client writes to the WAL before completing an update to Datamon. 

##  Goals
1. Allow an offline process to index and serve queries for all the metadata that is in Datamon.
2. A deterministic process to list every update to a Datamon instance.  
3. All successful writes are listed.
4. No centralized Datamon server in the data path

## Non Goals
1. Strict ordering of updates to Datamon instance
2. Recording failures in WAL

## Notes
The following points are used to establish the WAL correctness

### When is a write complete?

Entities in Datamon (bundles, repos, etc) when being created in Datamon can include multiple objects that need to be written.
A write is complete when the final descriptor object is written. Example: When a bundle is being uploaded, the upload is 
complete when the bundle descriptor file is written.

### KSUID

KSUID allow for kstorable unique IDs. 
[Timestamp can be used to generate a new KSUID](https://github.com/segmentio/ksuid/blob/master/ksuid.go#L213)

### Signed URL

[A signed URL allows for the expiration of the URL](https://cloud.google.com/storage/docs/access-control/signed-urls) 
There is an assumption here that Google/AWS does a better job at managing NTP for the servers backing the object store 
than the configuration of individual devices that write to Datamon. 
[Google manages NTP](https://cloud.google.com/compute/docs/instances/managing-instances)

### Timestamps on object updates
Coupled with the consistency of objects and updating metadata, google cloud API allow for a way to establish a last updated timestamp.

We cannot depend on the wall clock used by the Datamon SDK that can be running on an arbitrary device. 
Developers can upload data from their laptop that may or may not be using the correct time.

### Successful sequences

#### Sequence for a successful bundle write 

1. SDK uploads all the data blobs.
   1. On failure the deduplication offered by CAFS will insure that data is not rewritten.
2. SDK writes the intent to the WAL and gets a KSUID for the bundle.
   1. A bundle upload can still fail even though a record of the intent is written to the WAL.
   2. WAL generates the id that is guaranteed to be after a point in time in the WAL.
   3. WAL generates KSUID based on consistency of NTP time within GCS.
3. SDK gets an signed URL with expiration
   1. Expiration avoids the pitfalls of a asynchronous model and failure detection. If the URL expires as per GCS servers, 
   the write has failed. The SDK will now have to resume from step 2.
   2. The expiration period allows for a guarantee that the WAL is ordered and that no new writes will emerge in the WAL 
   before a certain KSUID in the lexical sorting of keys.
4. SDK writes the descriptor file completing the write.

#### Sequence for a successful repo creation

1. SDK writes the intent to the WAL
2. SDK gets a signed URL that will expire
3. SDK writes the descriptor completing the write.

#### Sequence for a successful label creation

1. SDK writes the intent to the WAL
2. SDK gets a signed URL that will expire
3. SDK writes the descriptor completing the write.
