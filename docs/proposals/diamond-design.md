# Diamond upload support

A diamond workflow allows for a pipeline to scale out the generation of a single bundle.
This relies on datamon's diamond uploads.

A diamond is composed of multiple splits; each split tries to write a set of files.
Ideally, 2 splits do not write the same file.

To elaborate, a bundle consists of a filesystem namespace with files in folders; each split can write files to any
folder but should not try to write the same file.

The individual nodes add files to splits without any co-ordination between them.

Commit logic handles any conflicts that exist. Conflicts can only occur if the same file in the namespace of
the bundle is written to by more than one split with different data.

Diamonds and splits are essentially transitory objects, which live in the metadata namespace of a datamon repo,
and can be deleted once completed. Just like for labels, these objects are not part of the "core" metadata
describing repos and bundles and are stored on an ancillary _versioned_ metadata bucket (`vmetadata`).

## Terminology

* **node**: a computer running an instance of datamon, e.g. a container or a pod
* **diamond**: an action performed in parallel by several independent nodes.
  In the remainder of this document, the action corresponds to uploading files as a datamon bundle.
* **split**: the portion of work in a diamond carried out by a single node
* **conflict**: a situation in which several splits upload overlapping files with different content
* **checkpoint**: another name for conflict, which denotes the user's intent to follow up overlapping files as versions of the same file
* **workflow**: in the context of this document, refers to a sequence of operations carried out by some collaborating nodes,
  not implying necessarily that this is an _ARGO_ workflow
* **incremental upload**: an upload operation conducted in several subsequent steps, resulting in a final commit as a bundle

## Workflow

The diamond workflow consists of the following steps.

### Diamond initialization

For a diamond to start, datamon needs some unique ID. Datamon SDK allows for the generation of such a unique ID for a diamond.

1. Prerequisite: generate a unique KSUID and a signaling metadata object that allows for starting splits.
2. All clients participating in the same diamond are expected to use the same ID at each step.
   **This is a requirement for client coordination**.
3. The {diamond-id} is only used once: users cannot initialize the same diamond twice.
4. Initialization command, returning an identifier which is guaranteed to be unique:
   ```
   datamon diamond initialize --repo {repo}
   {diamond-id}
   ```
5. Alternatively, a user can specify its ID for a diamond, which uniqueness is checked by datamon (this ID need not be a KSUID).
   ```
   datamon diamond initialize --repo {repo} --diamond {diamond-id}
   {diamond-id}
   ```
6. Backend model: the following path is created in the vmetadata bucket.
   ```
   /diamonds/{repo}/{diamond-id}/diamond-running.yaml
   ```

> **Trade-off**: the requirement for the initialization stage is set on purpose.
>
> This is not guided by implementation constraints but by the desire to protect users against unintentional silent data
> corruption, which could occur if the diamond ID were left as a user-controlled parameter.

## Concurrency model

The metadata model ensures that no metadata object is ever overwritten: some metadata may be wasted, but never be corrupted
by a concurrent operation.

As such, files are never locked: the only concurrency property required from the underlying storage is to be able to
ensure object creation in some bucket in an atomic way, with no allowed clobbering of its previous content.

### Files uploading

Uploading files in a diamond split first stores all files as blobs, then generates a file list in the versioned metadata bucket (`vmetadata`).

1. Files are uploaded by clients concurrently. This operation does not result in a new bundle yet.
   Files are in a pre-committed state, meaning that all content is persisted as blobs on the CAFS store, but not accessible from regular datamon
   commands yet.
2. Command:
   ```
   datamon diamond add --repo {repo} --diamond {diamond-id} --path {source}
   {split-id}
   ```
   This command is similar to `datamon bundle upload`, the only difference being the required `--diamond` flag.
3. Backend model: in the vmetadata bucket, the following paths will be created, and **never overwritten**.
   ```
   /diamonds/{repo}/{diamond-id}/splits/{split-id}/split-running.yaml
   /diamonds/{repo}/{diamond-id}/splits/{split-id}/{generation-id}/bundle-files-{index}.yaml
   ```
4. The split ID is internal to datamon, and automatically generated with every `add` command.
   The user is normally not required to know about this ID.
5. However, the user may run again a failed split by explicitly recalling the same `{split-id}`.
   This disables all conflict detection on files coming from that split. See more about conflicts [below](#handling-conflicts).
   ```
   datamon diamond add --repo {repo} --diamond {diamond-id} --split {split-id} --path {source}
   {split-id}
   ```
   This command fails when the split has not been already created.
6. Incremental uploads for a bundle and diamonds work the same way. See [below](#diamonds-vs-incremental-uploads)


> **Trade-off**: the possibility given to the user to opt-in for a specific `{split-id}` reflects the desire
> to avoid "technical conflicts" whenever needed.
>
> In all situations not requiring this kind of handling, not caring about this id is the recommended way.

> **generation-id**: this internal ID is allocated automatically and ensures that no action on splits can interfere
> with the file lists produced during the execution of a split, even when a user specifies the `{split-id}`.

### Commit as a bundle

Eventually, all file lists combine into a new bundle with the commit message and possibly a label.

1. The initiator of the diamond **needs to coordinate with the different clients to decide a commit** and create a new bundle.
2. Any `split add` operation still in progress at commit time will be ignored in the resulting bundle.
3. Command:
   ```
   datamon diamond commit --repo {repo} --diamond {diamond-id} --message {message} [--label {label}]
   ```
4. Optionally a commit can be set to fail on conflict.
   ```
   datamon diamond commit --repo {repo} --diamond {diamond-id} \
                          --message {message} --label {label} --no-conflicts
   ```
   In that case, the conflicting splits are indicated in the error message.
5. Alternatively, the whole diamond operation may be canceled (see [§ Cancellation](#handling-failures-retries-and-cancellations)).
   This allows for explicit reporting about the state of ongoing tasks as well as for some housekeeping of the
   `vmetadata` bucket (see below).


### Metadata housekeeping

**Nice to have**. A background job periodically cleans up metadata for all terminated diamonds and their splits.

1. Regular users and service accounts are not authorized to delete objects in buckets.
2. After a commit or cancel has been carried out, we may safely remove all diamond-related metata.
3. We don't want metadata buckets to grow and hold forever what is essentially temporary.
4. A privileged job is responsible for cleaning, for all repositories in a given context, all references to terminated
   diamonds and their splits in the vmetadata bucket.

> **NOTE**: while not critical, this task guards our `vmetadata` bucket against undesirable pollution from tracks of
> failed CI jobs, etc. We could run this on every week's end for instance.


## Example

```bash
repo="test-repo"
diamond_id=$(datamon diamond initialize --repo "${repo}")

datamon diamond add --repo "${repo}" --diamond "${diamond_id}" --path /datasource/part1 &
datamon diamond add --repo "${repo}" --diamond "${diamond_id}" --path /datasource/part2 &
datamon diamond add --repo "${repo}" --diamond "${diamond_id}" --path /datasource/part3 &
{split-id-1}
{split-id-2}
{split-id-3}

datamon diamond split list --repo "${repo}" --diamond "${diamond_id}"

datamon diamond commit --repo "${repo}" --diamond "${diamond_id}" --message "Diamond constructed"
{bundle-id}
```


## Detailed design

### Diamonds vs incremental uploads

Diamonds and incremental uploads are very much alike. However, these different use-cases come with some nuances
regarding how to handle conflicts.

Use-cases:

1. Diamond: a workflow spawns many pods to produce a single bundle as output.
2. Incremental uploads from a single client: a single pod eventually produces a bundle, but proceeds in several steps,
   with intermediate checkpoints, to persist the output.
3. Incremental uploads in parallel: many pods output data incrementally, and eventually produce a bundle.
   We don't want conflicts between increments but would like to track conflicts between nodes.

The main difference lies with concurrency and requirements for client coordination: a diamond carried out by a single
client _is_ an incremental upload.

Another key difference is how the user wants to deal with conflicts.

Therefore, we need to capture user intent. We add some additional CLI flags to specify how datamon should deal
with conflicts. See also [§ Handling Conflicts below](#handling-conflicts).


Available options at commit time:
```
datamon diamond commit --diamond {diamond-id} \
                      [--with-conflicts]   # <- this is the default - overlaps are reported as conflicts detected when merging splits

datamon diamond commit --diamond {diamond-id} \
                       --with-checkpoints   # <- indicates an incremental upload: "conflicts" are reported as subsequent versions of an uploaded file

datamon diamond commit --diamond {diamond-id} \
                       --no-conflicts       # <- forbids any overlap to take place at commit time

datamon diamond commit --diamond {diamond-id} \
                       --ignore-conflicts   # <- do not report about any conflict: intermediate versions
```

Available option at upload time:
```
datamon diamond add --diamond {diamond-id} \
                    --split {split-id} --path {source}   #  <- locally ignores conflicts with previous instance of
                                                         #     {split-id}, no change in behavior at commit time
```

Use `--no-conflicts` to guard incremental uploads against clobbering already uploaded files, whenever this is not desirable.

Flags determining the conflict-handling mode cannot be used jointly.

### Handling concurrency

Premises:

1. Several diamonds can run concurrently on the same repo, resulting in distinct bundles, without any interference
2. Within a single diamond, several `add` commands may run concurrently to contribute to a single bundle
3. Only one eventual commit operation may be carried out
4. Some `split add` commands may fail, some may complete, some may be subject to a retry


Design:

1. Concurrent diamonds.
  * A split may start **if and only if** the diamond metadata file is already existing in the vmetadata bucket.
  * The uniqueness of the diamond ID allows for several diamonds to run in parallel.
    Note that uniqueness is required only for any single repo.

2. Concurrent splits.
  * Each `split add` command runs with its unique split ID: metadata can be safely modified within this namespace.
  * `split add` commands reusing a previously allocated split ID produce new file lists that are unique to this run.
    Among all splits started with the same split ID, only the split that first reaches completion will be taken into account,
    all the other will eventually fail and their work wasted, without corrupting the state of the completed split.

3. Commit consistency.
  * The diamond is in effect concluded with the first commit associated with it. No further action is allowed on a committed diamond.
  * Any attempt to commit while another commit is in progress will result in only one of these operations to eventually succeed
  * Several concurrent attempts to commit the same diamond concurrently result in the same outcome, provided all splits have completed

4. See below [§ Handling failures](#handling-failures-retries-and-cancellations)

> **NOTE**: a _controlled_ failure occurs when some exception handling may be performed so the program exits gracefully
> (example: network outage).  _Uncontrolled_ failures are sudden interruptions: pod crash, program panic, ...
>
> Datamon does not explicitly distinguish between these different cases: a split is either "done" or "running", running
> meaning more precisely "not done", which implies that it may have failed and will never complete.


### Handling conflicts

Use-cases:

* Diamond use-case: a workflow deploys many splits, with some overlap in the resulting dataset.
  Overlapping identical files do not trigger any conflict. The user wants to be warned about overlaps with variations
  (conflicting versions).
* One pod in the workflow did restart: we don't want to report any conflict from that kind of event.
* Incremental upload: a single pod proceeds with the construction of a dataset in several steps. We keep the intermediate versions of
  all modified files as checkpoints.
* Incremental upload with diamond: a workflow deploys many splits on different pods, every single pod proceeding incrementally. Versions are checkpointed.


Premises: about detecting conflicts

1. A bundle has a single namespace for files
2. If _distinct_ splits upload a file with the same path but different data, a conflict is detected at commit time
3. Conflicting files are the different versions of the same path uploaded by splits
4. Identical files do not trigger conflicts (check the root hash)
5. The current arbitration rule is latest write wins (use nano sec timestamp here, not ksuid)


Premises: about what to do with conflicts

6. By default, discarded versions of files are kept for possible future reuse (e.g. fixing merge conflicts, retrieving clobbered content),
   unless the user tells datamon these may be safely dropped (i.e. use the `--ignore-conflicts` flag).
7. Details about conflicts and checkpoints are retained as part of the bundled dataset, in some "hidden folders"
8. When users explicitly mention their intent to carry out _incremental uploads_, conflicts are renamed as checkpoints,
   thus establishing a distinction between "good" and "bad" conflicts.
9. When users explicitly mention their intent to ignore conflicts, conflict reporting is discarded and older versions of files
   simply dropped at commit time.
10. The various conflict-handling flags _do not change_ the content of the produced bundle, save the special hidden
    folders used to keep track of conflicts/checkpoints


> **NOTE**: we don't want to upload blobs without keeping a track of the corresponding hashes
> (which would lead to unrecoverable blobs) unless explicitly being told to.
> That is precisely the meaning of the `--ignore-conflicts` flag.


Design:

* The latest version always wins: incremental uploads overwriting the same file with several versions always ends up with
  safe deconfliction.
* References to all conflicting files, save the winner, will be stored in a special path on the dataset available
  with the bundle, and may be readily downloaded:
  ```
  /.conflicts/{split-id}/{path}
  ```
  or:
  ```
  /.checkpoints/{split-id}/{path}
  ```
* Conflicts and checkpoints in these special locations won't be uploaded if the downloaded content is reused to create a new
  bundle (i.e. `/.conflicts`, `/.checkpoints` are ignored by uploads just as are the metadata in `/.datamon`).

> **NOTE**: conflict resolution is based on the _upload time_ of the file (not modification time on the dataset), and
> is resolved locally by the host running the split client.
>
> Therefore, hosts running splits should be synchronized.


### Handling failures, retries, and cancellations

Premises:

1. Since the commit action is user-driven, there must be no limit set on the delay between the completion of the splits
   and the final commit.
2. Some `add` operations might fail: the client has the opportunity to either _retry_ any individual split or _cancel_
   the whole diamond operation.
3. This is different from the `bundle upload` operation, which only succeeds or fails as a whole.
   Since we now leave it to the user to decide if and when to commit, there is a newly available choice to retry on
   individual failed splits.
4. Blobs put on storage resulting from split uploads cannot be relinquished. We accept the possibility of some wasted
   storage as a result.
5. Failure to carry out a commit is a special case, see [below](#failed-commit).
6. Explicit cancellation may be skipped and does not affect concurrent diamonds or diamonds initiated
   in the future. Explicit cancellation is only expected from well-behaved users and helps maintain orderly metadata.
   This is a consequence of the blob deduplication strategy (the same blob might be claimed by another file, so we can't
   mark it as something to be deleted).


Design:

* Diamonds abide by the following state-transition model:
  1. `initialized`
  2. `done`|`canceled` (terminal states)

* Splits:
  1. `running`
  2. `done` (terminal state)

> **NOTE**: the `canceled` state is not critical to the functionality and merely provides better
> monitoring capabilities as well as safe cleaning up.


#### Successful add operation
A split upload is marked as complete in the following metadata file:
```
/diamonds/{repo}/{diamond-id}/{split-id}/split-done.yaml   #  <- State pass to "done"
```

Any subsequent commit will only consider splits in the `done` state.


#### Failed add operation
Any failure on a `split add` operation will just leave the split in a "running state".
There is no update of the metadata to mark it explicitly as "failed".

```
/diamonds/{repo}/{diamond-id}/splits/{split-id}/split-running.yaml   #  <- State remains "running"
```

> **NOTE**: it is not possible to distinguish an aborted or hanging `add` operation from a merely long-running one.
> To recover from hangs/uncontrolled failures, one should start a new split, which is always a safe operation.


#### Retry add operation

Retrying is just starting a new split.

Users are never required to specify the split ID (though they may do so: see also [below](#controlling-the-split-id)).

There are 2 ways to do this:
1. Without specifying any split ID
2. With a previously allocated split ID, either provided by datamon **or by the user**


Option (1) is the simplest way to go: this treats the new split as an independent one, and the choice of the adequate
conflict handling strategy is deferred to commit time.

Option (2) requires the user to keep the split ID and reuse it: all conflicts between 2 runs with the same split ID are
simply ignored. Besides, this option allows for _removing_ some files in an incremental upload.


Failed or still running splits have not been marked in state `done` and will be ignored at commit time.

1. Any "done" split may be safely played again: datamon does nothing. The CLI warns the user but does not shoot any error.
2. Retries should normally be faster since some blobs have most likely been already uploaded:
   the deduplication will recognize that fact and skip these blobs (no change here to the current ways).
3. If on the other hand, the new run comes with some modified version of a file, this version will be kept
   and a new conflict will be detected. For workflows favoring the latter approach, we suggest committing with either the
   `--with-checkpoints` to report overwritten files as checkpoints (good) rather than conflicts (bad).
   Workflows that don't want to care about such details should use `--ignore-conflicts`.

##### Controlling the split ID

For some use-cases, it may be useful to let the user have explicit control over the `{split-id}`.

Example:

Let's suppose we have some containers or pods that restart automatically upon failures.

Let's assume too, that at every restart, the job is adding files with some unique content (e.g. a timestamp).

In this situation, we don't want to keep track of conflicts, or even consider this as "checkpoints".

The way to go here is to have the restarting container reuse the same, explicit `{split-id}`:
cross-pods conflict-handling strategy is preserved at commit time, but such local conflicts are ignored.

The container will have to persist the initial `{split-id}` returned by the command, then reuse it.

Example: reusing the splitID generated by datamon

```bash
# assume .split is mounted on some persistent volume
if [[ ! -f ${mounted}/.split ]] ; then
  # first time
  datamon diamond add --diamond ${diamond} --path ${source} > ${mounted}/.split
else
  # upon restarts
  datamon diamond add --diamond ${diamond} --split $(cat ${mounted}/.split) --path ${source}
fi
```

Example: using an arbitrary splitID provided by the user

```bash
user_split="some-unique-value-which-remains-stable-across-pod-restarts"
datamon diamond add --diamond ${diamond} --path ${source} --split ${user_split}
```

1. The command succeeds even though no split `{split-id}` was already existing.
2. The command fails whenever the split `{split-id}` is already `done`.
3. The command succeeds whenever the split `{split-id}` is already `running`: users must ensure that the split
   job is terminated. In the use case above, the failed container cannot possibly have the previous run with the split
   still running.
4. The restarted `split add` job scratches all previously constructed file lists in:
   ```
   diamonds/{diamond-id}/diamonds/{split-id}/bundle-files-{index}.yaml
   ```

#### Failed commit

Commit is a quick, metadata-only operation.

However, the sequence of internal operations is critical and must be protected against any loss of integrity.
A failed commit must never leave metadata in an unusable or corrupted state.

1. Work performed during commit is idempotent: any failed or interrupted commit can be restarted with the same eventual result.
2. Diamond is marked in the `done` state: the bundle has been created and is readily available for use.

Committing a bundle to metadata involves a critical section that _must_ be protected from concurrency:
since the metadata files to be written cannot be overwritten, any attempt to overwrite one of them will cause the competing 
commit operation to fail.

Constraints:
* Commits occur from the `initialized` state only
* Attempting a concurrent commit while in `done` or `canceled` states is always rejected.

##### Ensuring correctness for a commit operation

This is not related to diamonds and should apply to ordinary `bundle upload` just as well.

1. scan the splits in the done state and merge file lists: write the new file list.
   If a failure leaves us here, the bundle metadata is incomplete, and deemed not being created.
2. deal with conflicts/checkpoints: write these extra file list (<- same remark)
3. write the bundle file lists (index files)
4. write the `bundle.yaml` file

### Auditability & monitoring

Premises:

* I'd like the metadata to keep track of the details of any split operation, with a track of failed and successful
  split uploads
* This is useful information to trace back performance issues (e.g. one node took much more time than its siblings)
  or understand conflicts
* I'd like to report about the status of the various splits and get some feedback about failure or completion
* This is useful to make sure I can safely proceed with committing the work done, without necessarily requiring an exit
  status from the clients
* I'd like to distinguish the logs from every split when collecting all the logs on my cluster

Design:

* Bundle metadata are unaffected
* Listing & reporting proceeds by querying `vmetadata` (just like labels)
* A "tag" option is provided, which bears no functionality save tainting the logs with this tag


#### Lineage tracking

Premises:

* We assume that usually, all diamonds are initiated and run by the same contributor (e.g. some ARGO workflow with a runID)
* However, in the case of several contributors, each contributing some splits to the diamond,
  we should keep track of all of them in the list of contributors to that bundle

Design:

* Each split keeps track of its originating contributor
* This metadata is at first recorded in:
  ```
  /diamonds/{repo}/{diamond-id}/splits/{split-id}/split-done.yaml
  ```
* At commit time, we merge contributors into the resulting bundle:
  ```
  /bundles/{repo}/{bundle-id}/bundle.yaml
  ```

## Additional commands

### List all initiated diamonds on a repo
```
datamon diamond list --repo {repo}
```

Completed diamonds are available in this listing until they are eventually cleaned up by the background cleaner.

### List all ongoing splits for a diamond on a repo
```
datamon diamond split list --repo {repo} --diamond {diamond-id}
```

Report about the ID, status, and timings of every ongoing or failed split.
Completed splits are available in this listing until they are eventually cleaned up by the background cleaner.


### Report/inspect about conflicting files on a bundle
```
datamon bundle download --repo {repo} --bundle {bundle-id} --destination /my-data
find /my-data/.conflicts
find /my-data/.checkpoints
```

Since no intermediate state of the data is lost, users are free to inspect and possibly reuse these files.

Please be reminded here that files with special paths `/.conflicts` and `/.checkpoints` are filtered out by uploads.

## Recap of new commands or new options to existing commands

```
datamon diamond
          |_  initialize --repo {repo}
          |
          |_  commit --repo {repo} --diamond {diamond-id} --message {commit message} [--with-conflicts|--with-checkpoints|--ignore-conflicts|--no-conflicts] [--label]
          |
          |_  list --repo {repo}
          |
          |_  split
                |
                |_ add --repo {repo} --diamond {diamond-id} --path {source} [--split-id {split-id}] [--split-tag {my-bespoke-distinctive-tag}] [--files {list of files}]
                |
                |_ list --repo {repo} --diamond {diamond-id}
```


## Recap of the extended metadata layout

### vmetadata

```
diamonds/{repo}/{diamond-id}/diamond-running.yaml                          # <- captures the diamond with its initialized state
diamonds/{repo}/{diamond-id}/diamond-done.yaml                             # <- captures the diamond with a final state: done or canceled
diamonds/{repo}/{diamond-id}/splits/{split-id}/split-running.yaml          # <- captures the split state, plus holds information about the running split, eventually merged into metadata for bundle
diamonds/{repo}/{diamond-id}/splits/{split-id}/split-done.yaml             # <- captures the split state, plus holds information about the running split, eventually merged into metadata for bundle

diamonds/{repo}/{diamond-id}/splits/{split-id}/{generation-id}/bundle-files-{index}.yaml # <- file index for uploaded data, at a unique location for this generation
```

### metadata

No change.


## Nice to have / future

### Cancel a diamond
```
datamon diamond cancel --repo {repo} --diamond {diamond-id}
```

This immediately terminates the diamond. Any ongoing running split is not interrupted or waited for: its outcome will just remain ignored.


### Identify splits with some custom node identifier
```
datamon diamond split add --repo {repo} --path {source} --split-tag {my-bespoke-distinctive-tag}
```

Tags are kept in metadata and can be used when reporting about running splits or help later with conflict resolution.
Tags are used as a recognition mark when analyzing datamon log output over several nodes.

Tags are typically a hostname or something that helps the end-user identify a node in the diamond.


### Strong ordering
`add` commands can be strongly ordered based on a design similar to WAL (KSUIDs alone do not guarantee strong ordering, and local timestamps
are not fully satisfactory).
This is a nice to have improvement, which can be added later to the implementation.

### List files in one or all splits
```
datamon diamond split list files --repo {repo} --diamond {diamond-id} [--split-id {split-id}]
```
