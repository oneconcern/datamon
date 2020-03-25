**Version: dev**

## datamon bundle update

Update a downloaded bundle with a remote bundle.

### Synopsis

Update a downloaded bundle with a remote bundle.  --destination is a location previously passed to the `bundle download` command.

```
datamon bundle update [flags]
```

### Options

```
      --bundle string            The hash id for the bundle, if not specified the latest bundle will be used
      --concurrency-factor int   Heuristic on the amount of concurrency used by various operations.  Turn this value down to use less memory, increase for faster operations. (default 100)
      --destination string       The path to the download dir. Defaults to some random dir /tmp/datamon-mount-destination{xxxxx}
  -h, --help                     help for update
      --label string             The human-readable name of a label
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

* [datamon bundle](datamon_bundle.md)	 - Commands to manage bundles for a repo

