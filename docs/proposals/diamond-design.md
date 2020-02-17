# Diamond workflow support

A diamond workflow allows for a pipeline to scale out the generation of a single bundle.

A diamond is composed of multiple splits; each split tries to write a set of files.
Ideally, 2 splits do not write the same file.

To elaborate, a bundle consists of a filesystem namespace with files in folders; each split can write files to any
folder but should not try to write the same file.

The individual nodes add files to splits without any co-ordination between them.

Commit logic handles any conflicts that exist. Conflicts can only occur if the same file in the namespace of
the bundle is written to by more than one split with different data.

Diamonds and splits are essentially transitory objects, which live in the metadata namespace of a datamon repo,
and can be deleted once completed.


## Workflow

The diamond workflow consists of the following steps.

### Diamond initialization

For a diamond to start, datamon needs some unique ID. Datamon SDK allows for the generation of such a unique ID for a diamond.

1. Prerequisite: generate a unique KSUID and a signaling metadata object that allows for starting splits.
2. All clients participating in the same diamond are expected to use the same ID at each step.
   **This is a requirement on client coordination**.
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
   /diamonds/{repo}/{diamond-id}/diamond.yaml
   ```

> **Trade-off**: the requirement on the initialization stage is set on purpose.
>
> This is not guided by implementation constraints but by the desire to protect users against unintentional silent data
> corruption, which could occur if the diamond ID were left as a user-controlled parameter.

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
3. Backend model: in the vmetadata bucket, the following paths will be created or overwritten.
   ```
   /diamonds/{repo}/{diamond-id}/splits/{split-id}/split.yaml
   /diamonds/{repo}/{diamond-id}/splits/{split-id}/bundle-files-{index}.yaml
   ```
4. The split ID is internal to datamon, and automatically generated with every `add` command.
   The user is normally not required to know about this ID.
5. However, the user may run again a failed split by explicitly recalling the same `{split-id}`.
   This disables all conflict detection on files coming from that split. See more about conflicts [below](#handling-conflicts).
   ```
   datamon diamond add --repo {repo} --diamond {diamond-id} --split {split-id} --path {source}
   {split-id}
   ```
   This command fails when when the split has not been already created.
6. Incremental uploads for a bundle and diamonds work the same way. See [below](#diamonds-vs-incremental-uploads)


> **Trade-off**: the possibility given to the user to opt-in for a specific `{split-id}` reflects the desire
> to avoid "technical conflicts" whenever needed.
>
> In all situations not requiring this kind of handling, not caring about this id is the recommended way.

### Commit as a bundle

Eventually, all file lists combine into a new bundle with the commit message and possibly a label.

1. The initiator of the diamond **needs to coordinate with the different clients to decide a commit** and create a new bundle.
2. Any `split add` operation still in progress at commit time will be ignored in the resulting bundle.
3. Command:
   ```
   datamon diamond commit --repo {repo} --diamond {id} --message {message} [--label {label}]
   ```
4. Optionally a commit can be set to fail on conflict.
   ```
   datamon diamond commit --repo {repo} --diamond {diamond-id} \
                          --message {message} --label {label} --no-conflicts
   ```
   In that case, the conflicting splits are indicated in the error message.
5. **Nice to have**. Alternatively, the whole diamond operation may be canceled (see [§ Cancellation](#handling-failures-retries-and-cancellations)).
   This allows for some housekeeping of the `vmetadata` bucket (see below)


### Metadata housekeeping

**Nice to have**. A background job periodically cleans up metadata for all terminated diamonds and their splits.

1. Regular users and service accounts are not authorized to delete objects in buckets.
2. After a commit or cancel has been carried out, we may safely remove all diamond-related metata.
3. We don't want metadata buckets to grow and hold forever what is essentially temporary.
4. A privileged job is in responsible for cleaning, for all repositories in a given context, all references to terminated
   diamonds and their splits in the vmetadata bucket.

> **NOTE**: while not critical, this task guards our `vmetadata` bucket against undesirable pollution from tracks of
> failed CI jobs, etc. We could run this on every week's end for instance.


## Example

```bash
repo="test-repo"
diamond=$(datamon diamond initialize --repo "${repo}")

datamon diamond add --repo "${repo}" --diamond "${diamond-id}" --path /datasource/part1 &
datamon diamond add --repo "${repo}" --diamond "${diamond-id}" --path /datasource/part2 &
datamon diamond add --repo "${repo}" --diamond "${diamond-id}" --path /datasource/part3 &

datamon diamond split list --repo "${repo}" --diamond "${diamond-id}"

datamon diamond commit --repo "${repo}" --diamond "${diamond-id}" --message "Diamond constructed"
```


## Detailed design

### Diamonds vs incremental uploads

Diamonds and incremental uploads are very much alike, the use-cases come with some nuances.

Use-cases:

1. Diamond: a workflow spawns many pods to produce a single bundle as output.
2. Incremental uploads from a single client: a single pod eventually produces a bundle, but proceeds in several steps,
   with intermediate checkpoints, to persist the output.
3. Incremental uploads in parallel: many pods output data incrementally, and eventually produce a bundle.
   We don't want conflicts between increments but would like to track conflicts between nodes.

The main difference lies with concurrency and requirements on client coordination: a diamond carried out by a single
client _is_ an incremental upload. The other difference is how the user wants to deal with conflicts.

Therefore, we need to capture user intent. We add some additional CLI flags to specify how datamon should deal
with conflicts. See also [§ Handling Conflicts below](#handling-conflicts).


Available options at commit time:
```
datamon diamond commit --diamond {diamond-id} \
                      [--with-conflicts]   # <- this is the default - overlaps are reported as conflicts detected when merging splits

datamon diamond commit --diamond {diamond-id} \
                       --with-checkpoints   # <- indicates an incremental upload: conflicts are subsequent versions of uploaded files

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

1. Several diamonds can run concurrently on the same repo, resulting in distinct bundles
2. Within a single diamond, several `add` commands may run concurrently to contribute to a single same bundle
3. Only one eventual commit operation may be carried out. Any attempt to compete on committing a diamond must be rejected.
4. Some `split add` commands may fail, some may complete, some may be subject to a retry


Design:

1. Concurrent diamonds.
  * A split may start **if and only if** the diamond metadata file is already existing in the vmetadata bucket.
  * The uniqueness of the diamond ID allows for several diamonds to run in parallel.
    Note that uniqueness is required only for any single repo.

2. Concurrent splits.
  * Each `split add` command runs with its uniquely generated internal ID: metadata can be safely modified within this namespace.

3. Commit consistency.
  * The diamond is in effect concluded with the first commit associated with it. No further action is allowed on a committed diamond.
  * Any attempt to commit while another commit is in progress is rejected (the operation should normally be fast
    since only metadata is affected).
  * For the duration of the commit operation, the diamond is in effect locked in "committing" status.
  * To recover from uncontrolled failures during a commit (e.g. pod crash,...) this lock is considered stale after some
    timeout (e.g. 30 sec) and a new commit may be attempted.

4. See below [§ Handling failures](#handling-failures-retries-and-cancellations)

> **NOTE**: a _controlled_ failure occurs when some exception handling may be performed so the program exits gracefully
> (example: network outage).
> _Uncontrolled_ failures are sudden interruptions: pod crash, program panic, ...
>
> Establishing a distinction between an explicitly failed status and some other kind of failures helps to report about
> the status of ongoing splits. It also helps to clean metadata.


### Handling conflicts

Use-cases:

* Diamond use-case: a workflow deploys many splits, with some overlap in the resulting dataset.
  Overlapping identical files do not trigger any conflict. The user wants to be warned about overlaps with differences
  (conflicting versions).
* One pod in the workflow did restart: we don't want to report any conflict from that kind of event.
* Incremental upload: a single pod proceeds to dataset construction in several steps. We keep intermediate versions of
  files as checkpoints.
* Incremental upload with diamond: a workflow deploys many splits, each single pod proceeds to dataset construction in
  several steps. Versions are checkpointed.


Premises: about detecting conflicts

1. A bundle has a single namespace for files
2. If _distinct_ splits upload a file with the same path but different data, a conflict is detected at commit time
3. Conflicting files are the different versions of the same path uploaded by splits
4. Identical files do not trigger conflicts (check the root hash)
5. The current arbitration rule is latest write wins (use nano sec timestamp here, not ksuid)


Premises: about what to do with conflicts

6. By default, discarded versions of files are kept for possible future reuse (e.g. fixing merge conflicts),
   unless the user tells datamon these may be safely dropped.
7. Details about conflicts and checkpoints are retained as part of the bundled dataset, in some "hidden folders"
8. When users explicitly mention their intent to carry out _incremental uploads_, conflicts are renamed as checkpoints,
   thus establishing a distinction between "good" and "bad" conflicts.
9. When users explicitly mention their intent to ignore conflicts, conflicts are discarded and older versions of files
   simply dropped at commit time.

> **NOTE**: we don't want to upload blobs without keeping a track of the reference hashes
> (unrecoverable blobs), unless explicitly being told to.


Design:

* Latest version wins: incremental uploads overwriting the same file with several versions always ends up with a
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
* Conflicts and checkpoints in these special locations won't be uploaded if the download is reused to create a new
  bundle (i.e. `/.conflicts`, `/.checkpoints` are ignored by uploads just as are the metadata in `.datamon`).

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

6. **Nice to have**. Explicit cancellation may be skipped and does not affect concurrent diamonds or diamonds initiated
   in the future. Explicit cancellation is only expected from well-behaved users and helps maintain orderly metadata.
   This is a consequence of the blob deduplication strategy (the same blob might be claimed by another file, so we can't
   mark it as something to be deleted).


Design:

* Diamonds abide by the following state-transition model:
  1. `initialized`
  2. `committing`
  3. `done`

* **Nice to have**: besides, for diamond canceling operations:
  4. `canceling`
  5. `canceled`

* Splits:
  1. `running`
  2. `done`|`failed`

> **NOTE**: the `canceled` and `failed` states are not critical to the functionality: they merely provide better
> monitoring capabilities ("what's going on with my splits?") as well as safe cleaning up.


#### Successful add operation
A split upload is marked as complete in the following metadata file:
```
/diamonds/{repo}/{diamond-id}/{split-id}/split.yaml   #  <- State pass to "done"
```

Any subsequent commit will only consider splits in the `done` state.


#### Failed add operation
A controlled failure on such an operation would mark the split as explicitly failed (for further reporting and monitoring).
```
/diamonds/{repo}/{diamond-id}/splits/{split-id}/split.yaml   #  <- State pass to "failed"
```

> **NOTE**: please mark that it is not possible to distinguish an aborted or hanging `add` operation from a merely long-running one.
> To recover from hangs/uncontrolled failures, one should start a new split.


#### Retry add operation

Retrying is just starting a new split. Users are never required to specify the split ID (though they may: see
[below](#controlling-the-split-id)). Failed splits have not been marked in state `done` and will be ignored at commit
time.  This metadata will eventually be cleaned up.

1. If a "done" split is played again, there should be no impact: all uploaded files being equal,
   no conflict shall be triggered.
2. Retries should normally be faster since some blobs have most likely been already uploaded:
   the deduplication will recognize that fact and skip these blobs (no change here to the current ways).
3. If on the other hand, this new run comes with some modified version of a file, this version will be kept
   and a new conflict will be detected. For workflows favoring the latter approach, we suggest committing with either the
   `--with-checkpoints` to report overwritten files as checkpoints (good) rather than conflicts (bad).
   Workflows that don't want to care about such details should use `--ignore-conflicts`.

##### Controlling the split ID

For some use-cases, it may be useful to let the user have explicit control over the `{split-id}`.

Example:

Let's suppose we have some containers or pods that restart automatically upon failures.

Let's assume too, that at every restart, the job is adding files with some unique content (e.g. a timestamp).

In this situation, we don't want to keep track of conflicts, or even consider this as "checkpoints".

The way to go here is to have the restarting container always restart with the same, explicit `{split-id}`:
overall conflict-handling is preserved at commit time, but such technical conflicts are ignored.

The container will have to persist the initial `{split-id}` returned by the command, then reuse it.

Example:

```bash
# assume .split is mounted on some persistent volume
if [[ ! -f ${mounted}/.split ]] ; then
  datamon diamond add --diamond ${diamond} --path ${source} > ${mounted}/.split
else
  datamon diamond add --diamond ${diamond} --split $(cat ${mounted}/.split) --path ${source}
fi
```

1. The command fails whenever no split `{split-id}` is not already existing.
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
2. Controlled failure (managed error): the locked status on the diamond metadata file
   (`diamond.yaml  #  <- State: "committing"`) moves the state back to "initialized" and a new commit operation may
   be attempted straight away.
3. Uncontrolled failure (container failure, panic...): the locked status could not be updated, but will expire.
   Commit can be restarted after waiting for the lock to expire


Committing a bundle to metadata involves a critical section that _must_ be protected from concurrency.

1. From aggregation of splits down to bundle metadata creation. Diamond is marked in the `committing` state.
2. The bundle has been created and is readily available for use. Diamond is marked in the `done` state.

Constraints:
* Under normal conditions, commit occurs from the `initialized` state
* Attempting a concurrent commit while in `committing` fresh state is rejected
* Attempting a concurrent commit while in `committing` stale state, the lock timestamp is refreshed and commit may proceed
* Before proceeding with finalizing the commit (i.e. `PUT bundle.yaml`), a last-ditch check on the lock timestamp is
  performed: if it has been updated, the commit fails.
* Attempting a concurrent commit while in `done` is always rejected.
* When eventually cleaned up by the background job, the vmetadata no more hold any reference to the diamond.

Technically, there is indeed a race condition on updating `diamond.yaml` twice. The last-ditch check on a nano sec
timestamp ensures that only the latest started commit eventually completes (with its snapshot of done splits).

##### Ensuring correctness for a commit operation

This is not related to diamonds and should apply to ordinary `bundle upload` just as well.

1. scan the splits in the done state and merge file lists: write the new file list
   (<- if a failure leaves us here, the bundle metadata is incomplete, and deemed not being created)
2. deal with conflicts/checkpoints: write these extra file list (<- same remark)
3. write the `bundle.yaml` file

### Auditability & monitoring

Premises:

* I'd like the metadata to keep track of the details of any split operation, with a track of failed and successful
  split uploads
* This is useful information to trace back performance issues (e.g. one of the node took much more time than its siblings)
  or understand conflicts
* I'd like to report about the status of the various splits and get some feedback about failure or completion
* This is useful to make sure I can safely proceed with committing the work done, without necessarily requiring an exit
  status from the clients

Design:

* Bundle metadata are unaffected
* Listing & reporting proceeds by querying `vmetadata` (just like labels)


#### Lineage tracking

Premises:

* We assume that usually, all diamonds are initiated and run by the same contributor (e.g. some ARGO workflow with a runID)
* However, in the case of several contributors, each contributing some splits to the diamond,
  we should keep track of all of them in the list of contributors to that bundle

Design:

* Each split keeps track of its originating contributor
* This metadata is at first recorded in:
  ```
  /diamonds/{repo}/{diamond-id}/splits/{split-id}/split.yaml
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
diamonds/{repo}/{diamond-id}/diamond.yaml                                  # <- captures the diamond state: initialized, committing, done
diamonds/{repo}/{diamond-id}/diamonds/{split-id}/split.yaml                # <- captures the split state, plus holds information about the running split, eventually merged into metadata for bundle
diamonds/{repo}/{diamond-id}/diamonds/{split-id}/bundle-files-{index}.yaml # <- file index for uploaded data
```

### metadata

No change.


## Nice to have / future

### Cancel a diamond
```
datamon diamond cancel --repo {repo} --diamond {diamond-id}
```

This waits for ongoing running splits to complete then marks vmetadata as ready for some cleanup

> **NOTE**: in the future, we may interrupt running splits rather than waiting for them to complete _then_ cancel the diamond


### Identify splits with some custom node identifier
```
datamon diamond split add --repo {repo} --path {source} --split-tag {my-bespoke-distinctive-tag}
```

Tags are kept in metadata and can be used when reporting about running splits or help later with conflict resolution.
Tags are typically a hostname or something that helps the end-user identify a node in the diamond.

   1. `add` commands can be strongly ordered based on a design similar to WAL (KSUIDs alone do not guarantee strong ordering, and local timestamps
      are not fully satisfactory).
      This is a nice to have addition and can be added later in the implementation.
   2. explicit cancellation could attempt to interrupt ongoing splits


### List files in one or all splits
```
datamon diamond split list files --repo {repo} --diamond {diamond-id} [--split-id {split-id}]
```
