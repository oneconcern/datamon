**Version: dev**

## datamon bundle get

Get bundle info

### Synopsis

Performs a direct lookup of a bundle.

Prints corresponding bundle metadata if the bundle exists,
exits with ENOENT status otherwise.

```
datamon bundle get [flags]
```

### Options

```
      --bundle string     The hash id for the bundle, if not specified the latest bundle will be used
  -h, --help              help for get
      --label string      The human-readable name of a label
      --repo (*) string   The name of this repository
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
      --skip-auth                 Skip authentication against google (gcs credentials remains required)
      --upgrade                   Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon bundle](datamon_bundle.md)	 - Commands to manage bundles for a repo

