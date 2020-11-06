**Version: dev**

## datamon label list

List labels

### Synopsis

List the labels in a repo.

This is analogous to the "git tag --list" command.

```
datamon label list [flags]
```

### Examples

```
% datamon label list --repo ritesh-test-repo
init , 1INzQ5TV4vAAfU2PbRFgPfnzEwR , 2019-03-12 22:10:24.159704 -0700 PDT
```

### Options

```
      --batch-size int           Number of bundles streamed together as a batch. This can be tuned for performance based on network connectivity (default 1024)
      --concurrency-factor int   Heuristic on the amount of concurrency used by core operations. Concurrent retrieval of metadata is capped by the 'batch-size' parameter. Turn this value down to use less memory, increase for faster operations. (default 500)
  -h, --help                     help for list
      --prefix string            List labels starting with a prefix.
      --repo (*) string          The name of this repository
      --with-versions            List all previous versions of labels
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
      --upgrade                   Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon label](datamon_label.md)	 - Commands to manage labels for a repo

