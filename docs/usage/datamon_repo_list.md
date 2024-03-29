**Version: dev**

## datamon repo list

List repos

### Synopsis

List repos that have been created

```
datamon repo list [flags]
```

### Examples

```
% datamon repo list --context ctx2
fred , test fred , Frédéric Bidon , frederic@oneconcern.com , 2019-12-05 14:01:18.181535 +0100 CET
```

### Options

```
      --batch-size int           Number of bundles streamed together as a batch. This can be tuned for performance based on network connectivity (default 1024)
      --concurrency-factor int   Heuristic on the amount of concurrency used by core operations. Concurrent retrieval of metadata is capped by the 'batch-size' parameter. Turn this value down to use less memory, increase for faster operations. (default 500)
  -h, --help                     help for list
      --skip-auth                Skip authentication against google (gcs credentials remains required)
      --with-size                Reports the assessed repo size in bytes for all bundles, without accounting for deduplicated blobs
```

### Options inherited from parent commands

```
      --config string             Set the config backend store to use (bucket name: do not set the scheme, e.g. 'gs://')
      --context string            Set the context for datamon (default "dev")
      --format string             Pretty-print datamon objects using a Go template. Use '{{ printf "%#v" . }}' to explore available fields
      --loglevel string           The logging level. Levels by increasing order of verbosity: none, error, warn, info, debug (default "info")
      --metrics                   Toggle telemetry and metrics collection
      --metrics-password string   Password to connect to the metrics collector backend. Overrides any password set in URL
      --metrics-url string        Fully qualified URL to an influxdb metrics collector, with optional user and password
      --metrics-user string       User to connect to the metrics collector backend. Overrides any user set in URL
      --upgrade                   Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon repo](datamon_repo.md)	 - Commands to manage repos

