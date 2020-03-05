**Version: dev**

## datamon diamond split

Commands to manage splits

### Synopsis

A split is a part of a diamond, which may be used to upload data concurrently

### Options

```
  -h, --help   help for split
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

* [datamon diamond](datamon_diamond.md)	 - Commands to manage diamonds
* [datamon diamond split add](datamon_diamond_split_add.md)	 - adds a new split and starts uploading
* [datamon diamond split get](datamon_diamond_split_get.md)	 - Gets split info
* [datamon diamond split list](datamon_diamond_split_list.md)	 - Lists splits in a diamond and in a repo

