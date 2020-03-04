**Version: dev**

## datamon upgrade

Upgrades datamon to the latest release

### Synopsis

Checks for the latest release on github repo then upgrades. By default upgrade is skipped if the current datamon is not a released version

```
datamon upgrade [flags]
```

### Options

```
      --check-version   Checks if a new version is available but does not upgrade
      --force           Forces upgrade even if the current version is not a released version
      --format string   Pretty-print datamon objects using a Go template. Use '{{ printf "%#v" . }}' to explore available fields
  -h, --help            help for upgrade
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

