**Version: dev**

## datamon purge

Commands to purge unused blob storage

### Synopsis

Purge allows owners of a BLOB storage to actually delete data that is no longer referenced by any repo.

To effectively proceed to a purge, proceed with the following steps:
1. Use "datamon repo delete" to delete repositories. This will remove references to a repo. Actual BLOB storage is maintained.
2. Use "datamon purge build-reverse-lookup". This will build an index all currently active BLOB references for _all_ repositories.
3. Use "datamon purge delete-unused". This will delete BLOB resources that are not present in the index.

NOTES:
* datamon purge delete-unused-blobs won't start if no reverse-lookup index is present
* datamon purge build-reverse-lookup may be run again, thus updating the index
* the update time considered for the reverse-lookup index is the time the build command is launched
* any repo or file object that is created while building the index will be ignored in the index.
* when running delete-unused, BLOB pages that are more recent than the index won't be removed.


### Options

```
      --force                   Forces a locked purge job to run. You MUST make sure that no such concurrent job is running
  -h, --help                    help for purge
      --local-work-dir string   Indicates the local folder that datamon will use as its working area (default ".datamon-index")
      --skip-auth               Skip authentication against google (gcs credentials remains required)
```

### Options inherited from parent commands

```
      --config string             Set the config backend store to use (bucket name: do not set the scheme, e.g. 'gs://')
      --context string            Set the context for datamon (default "dev")
      --loglevel string           The logging level. Levels by increasing order of verbosity: none, error, warn, info, debug (default "info")
      --metrics                   Toggle telemetry and metrics collection
      --metrics-password string   Password to connect to the metrics collector backend. Overrides any password set in URL
      --metrics-url string        Fully qualified URL to an influxdb metrics collector, with optional user and password
      --metrics-user string       User to connect to the metrics collector backend. Overrides any user set in URL
      --upgrade                   Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon](datamon.md)	 - Datamon helps build ML pipelines
* [datamon purge build-reverse-lookup](datamon_purge_build-reverse-lookup.md)	 - Command to build a reverse-lookup index of used BLOB resources
* [datamon purge delete-reverse-lookup](datamon_purge_delete-reverse-lookup.md)	 - Command to delete a reverse-lookup index from the metadata
* [datamon purge delete-unused](datamon_purge_delete-unused.md)	 - Command to delete BLOB resources that are not present in the reverse-lookup index

