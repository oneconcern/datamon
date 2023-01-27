**Version: dev**

## datamon purge build-reverse-lookup

Command to build a reverse-lookup index of used BLOB resources

### Synopsis

The index may be updated, unless a delete-unused command is currently running.

Only ONE instance of this command may run: concurrent index building is not supported.

If a build-reverse-lookup OR delete-unused command was running and failed, an update of the index may be forced using the "--force" flag.

You MUST make sure that no concurrent build-reverse-lookup or delete job is still running before doing that.


```
datamon purge build-reverse-lookup [flags]
```

### Options

```
      --concurrency-factor int   Heuristic on the amount of concurrency used by various operations.  Turn this value down to use less memory, increase for faster operations. (default 100)
      --current-context-only     Index building is only applied to the metadata of the current context
  -h, --help                     help for build-reverse-lookup
      --index-chunk-start int    Index building starts with this index chunk sequence number. This allows for manually copying other chunks and merging indexes
      --resume                   Resume index building: reload already uploaded index files (implies --force)
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

