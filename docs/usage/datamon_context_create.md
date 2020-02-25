**Version: dev**

## datamon context create

Create a context

### Synopsis

Create a context for Datamon

```
datamon context create [flags]
```

### Options

```
      --blob string       The name of the bucket hosting the datamon blobs
  -h, --help              help for create
      --meta string       The name of the bucket used by datamon metadata
      --read-log string   The name of the bucket hosting the read log
      --vmeta string      The name of the bucket hosting the versioned metadata
      --wal string        The name of the bucket hosting the WAL
```

### Options inherited from parent commands

```
      --config string        Set the config backend store to use (bucket name: do not set the scheme, e.g. 'gs://')
      --context string       Set the context for datamon (default "dev")
      --format string        Pretty-print datamon objects using a Go template. Use '{{ printf "%#v" . }}' to explore available fields
      --loglevel string      The logging level. Levels by increasing order of verbosity: none, error, warn, info, debug (default "info")
      --metrics              Toggle telemetry and metrics collection
      --metrics-url string   Fully qualified URL to an influxdb metrics collector, with user and password
      --upgrade              Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon context](datamon_context.md)	 - Commands to manage contexts.

