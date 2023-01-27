**Version: dev**

## datamon purge delete-unused

Command to delete BLOB resources that are not present in the reverse-lookup index

### Synopsis

The reverse-lookup index MUST have been created.

Any BLOB resource that is more recent than the index last update date is kept.

Only ONE instance of this command may run: concurrent deletion is not supported.
Index updates cannot be performed while the deletion is ongoing.

If the delete-unused job fais to complete, it may be run again.

To retry on a failed deletion, use the "--force" flag to bypass the lock.
You MUST make sure that no delete job is still running before doing that.


```
datamon purge delete-unused [flags]
```

### Options

```
      --concurrency-factor int   Heuristic on the amount of concurrency used by various operations.  Turn this value down to use less memory, increase for faster operations. (default 100)
      --dry-run                  Report about the purge, but don't actually delete anything
  -h, --help                     help for delete-unused
```

### Options inherited from parent commands

```
      --config string             Set the config backend store to use (bucket name: do not set the scheme, e.g. 'gs://')
      --context string            Set the context for datamon (default "dev")
      --force                     Forces a locked purge job to run. You MUST make sure that no such concurrent job is running
      --local-work-dir string     Indicates the local folder that datamon will use as its working area (default ".datamon-index")
      --loglevel string           The logging level. Levels by increasing order of verbosity: none, error, warn, info, debug (default "info")
      --metrics                   Toggle telemetry and metrics collection
      --metrics-password string   Password to connect to the metrics collector backend. Overrides any password set in URL
      --metrics-url string        Fully qualified URL to an influxdb metrics collector, with optional user and password
      --metrics-user string       User to connect to the metrics collector backend. Overrides any user set in URL
      --skip-auth                 Skip authentication against google (gcs credentials remains required)
      --upgrade                   Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon purge](datamon_purge.md)	 - Commands to purge unused blob storage

