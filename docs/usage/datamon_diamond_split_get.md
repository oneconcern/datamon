**Version: dev**

## datamon diamond split get

Gets split info

### Synopsis

Performs a direct lookup of a split.

Prints corresponding split metadata if the split exists,
exits with ENOENT status otherwise.

```
datamon diamond split get [flags]
```

### Options

```
      --diamond string   The diamond to use
  -h, --help             help for get
      --repo string      The name of this repository
      --split string     The split to use
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

* [datamon diamond split](datamon_diamond_split.md)	 - Commands to manage splits

