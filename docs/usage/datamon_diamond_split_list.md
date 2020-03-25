**Version: dev**

## datamon diamond split list

Lists splits in a diamond and in a repo

### Synopsis

Lists splits in a diamond and in a repo, ordered by their start time

```
datamon diamond split list [flags]
```

### Options

```
      --batch-size int           Number of bundles streamed together as a batch. This can be tuned for performance based on network connectivity (default 1024)
      --concurrency-factor int   Heuristic on the amount of concurrency used by core operations. Concurrent retrieval of metadata is capped by the 'batch-size' parameter. Turn this value down to use less memory, increase for faster operations. (default 500)
      --diamond string           The diamond to use
  -h, --help                     help for list
      --repo string              The name of this repository
```

### Options inherited from parent commands

```
      --config string             Set the config backend store to use (bucket name: do not set the scheme, e.g. 'gs://')
      --context string            Set the context for datamon (defaults to "dev")
      --format string             Pretty-print datamon objects using a Go template. Use '{{ printf "%#v" . }}' to explore available fields
      --loglevel string           The logging level. Levels by increasing order of verbosity: none, error, warn, info, debug (default "info")
      --metrics                   Toggle telemetry and metrics collection
      --metrics-password string   Password to connect to the metrics collector backend. Overrides any password set in URL
      --metrics-url string        Fully qualified URL to an influxdb metrics collector, with optional user and password
      --metrics-user string       User to connect to the metrics collector backend. Overrides any user set in URL
      --upgrade                   Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon diamond split](datamon_diamond_split.md)	 - Commands to manage splits

