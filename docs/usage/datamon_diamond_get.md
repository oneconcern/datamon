**Version: dev**

## datamon diamond get

Gets diamond info

### Synopsis

Performs a direct lookup of a diamond.

Prints corresponding diamond metadata if the diamond exists,
exits with ENOENT status otherwise.

```
datamon diamond get [flags]
```

### Options

```
      --diamond string   The diamond to use
  -h, --help             help for get
      --repo string      The name of this repository
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

* [datamon diamond](datamon_diamond.md)	 - Commands to manage diamonds

