**Version: dev**

## datamon web

Webserver

### Synopsis

A webserver process to browse datamon data

```
datamon web [flags]
```

### Options

```
  -h, --help         help for web
      --no-browser   Disable automatic launch of a browser
      --port int     Port number for the web server (defaults to random port)
```

### Options inherited from parent commands

```
      --config string        Set the config backend store to use (bucket name: do not set the scheme, e.g. 'gs://')
      --context string       Set the context for datamon (default "dev")
      --loglevel string      The logging level. Levels by increasing order of verbosity: none, error, warn, info, debug (default "info")
      --metrics              Toggle telemetry and metrics collection
      --metrics-url string   Fully qualified URL to an influxdb metrics collector, with user and password
      --upgrade              Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon](datamon.md)	 - Datamon helps build ML pipelines

