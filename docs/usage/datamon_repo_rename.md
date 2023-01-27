**Version: dev**

## datamon repo rename

Rename a repo

### Synopsis

Rename an existing datamon repository.

You must authenticate to perform this operation (can't --skip-auth).
You must specify the context with --context.

This command MUST NOT BE RUN concurrently.


```
datamon repo rename {new repo name} [flags]
```

### Examples

```
% datamon repo rename --context dev --repo ritesh-datamon-test-repo ritesh-datamon-new-repo
```

### Options

```
      --context (*) string   Set the context for datamon (default "dev")
      --force-yes            Bypass confirmation step
  -h, --help                 help for rename
      --repo (*) string      The name of this repository
```

### Options inherited from parent commands

```
      --config string             Set the config backend store to use (bucket name: do not set the scheme, e.g. 'gs://')
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

