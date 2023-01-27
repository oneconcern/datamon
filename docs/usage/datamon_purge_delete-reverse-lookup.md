**Version: dev**

## datamon purge delete-reverse-lookup

Command to delete a reverse-lookup index from the metadata

### Synopsis

The index maybe quite large and only really used when we need to purge BLOBs.

This command allows to remove the index file from the metadata.
Only ONE instance of this command may run: dropping index concurrently is not supported.

A deletion of the index may be forced using the "--force" flag.

You MUST make sure that no concurrent build-reverse-lookup or delete job is still running before doing that.


```
datamon purge delete-reverse-lookup [flags]
```

### Options

```
  -h, --help   help for delete-reverse-lookup
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

