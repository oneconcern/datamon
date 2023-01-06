# Purging deleted data

`datamon` is essentially an immutable store: it is not designed to delete stored resources.

However, we have added the capability to occasionally trim the ever-growing amount of storage,
by removing metadata and stored BLOBs that are not in use.

Since BLOBs are deduplicated, a single BLOB resource may be referenced by one or several files from different repos.

## How to proceed

1. Delete unnecessary data
There are a few ways to derefence the data that you no longer want to keep.
* delete a repo (`datamon repo delete`)
* delete some files in all bundles from a repo (`datamon repo delete files`)
* squash a repo to retain only some bundles (`datamon repo squash`)

1.1 Delete the repos that you no longer want
```
datamon repo delete --repo {my-deprecated-repo} --context dev
```

This will remove the metadata for this repo: all bundles and files are now irrecoverable.
However, raw BLOB storage is still there (because we don't know yet if other resources refer to them).

This command is very fast as it operates only on metadata.

1.2 Delete files in a repo
```
datamon repo delete files --repo {my-deprecated-repo} {--files file-list.txt|--file path/file-to-delete} --context dev
```

1.3 Squash bundles in a repo
This removes from the metadata some unwanted past versions of your data.

* Retain only the latest bundle:
```
datamon repo squash --repo {my-deprecated-repo} --context dev
```
* Retain the latest bundle AND all bundles with a tag (aka label):
```
datamon repo squash --repo {my-deprecated-repo} --retain-tags --context dev
```
* Retain the latest bundle AND all bundles with a valid semver tag (e.g. 1.2.3 or v1.2.3):
```
datamon repo squash --repo {my-deprecated-repo} --retain-semver-tags --context dev
```
* Retain the latest n bundles (possibly mixed with the retain tag options above)
```
datamon repo squash --repo {my-deprecated-repo} --retain-n-latest 10 --context dev
```

2. Build a reverse-lookup index of all BLOB keys currently in use on your blob bucket
```
datamon purge build-reverse-lookup [--context dev]
```

This will require some time and some local storage to store the keys: all bundles in all repos are scanned for a given context.

NOTE: running again the command will scratch the existing index and create an updated version.

NOTE: the index is by default built in `./.datamon-index` in the current working directory. This location may be altered
by using the `--local-work-dir` flag to the `datamon purge` commands.

> A kubernetes job template is provided in the folder `k8s/purge/build-index`.
> You can run it with a command such as:
>
> `helm install -n flood -f values.default.yaml .`

> **NOTE**: the job indexing keys in use may be interrupted and restarted with the `--resume` flag: it will reload the index 
> chunks already uploaded and resume indexing.

3. Delete all unused blobs
```
datamon purge delete-unused [--context dev]
```

This will remove permanently all BLOB keys that are not referenced by the index.
Again, for large stores, this command may take quite some time, as it is scanning all keys in the BLOB bucket.
Similarly, enough local storage must be added to store locally the index of used keys (e.g. ~ 10GB).

> A kubernetes job template is provided in the folder `k8s/purge/delete-unused`.
> You can run it with a command such as:
>
> `helm install -n flood -f values.default.yaml .`

4. You might want now to drop the index from the metadata
```
datamon purge delete-reverse-lookup [--context dev]
```

> **NOTE**: the job deleting BLOB keys may be interrupted and restarted: it will reload its index and 
> resume deletion.

## Caveats

All the `purge` commands MUST be run only by 1 process at a given time: there is an exclusive lock
created to prevent other such commands to run in parallel.

If the job fails for some reason and cannot remove the lock file, use the `--force` option for your retry command.
This will override the lock.

The `delete-unused` command only deletes BLOB files that are older than the index creation time.
If some new repos or bundles have been added between the launch of the `build-reverse-lookup` command
and the execution of the `delete-unused` command, the corresponding new BLOB items _won't be deleted_.

In order to delete such new files, you need to update the index using `purge build-reverse-lookup`.
