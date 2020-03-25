**Version: dev**

## datamon bundle download file

Download a file from bundle

### Synopsis

Download a readonly, non-interactive view of a single file
from a bundle.

You may use the "--label" flag as an alternate way to specify a particular bundle.


```
datamon bundle download file [flags]
```

### Examples

```
% datamon bundle download file --file datamon/cmd/repo_list.go --repo ritesh-test-repo --bundle 1ISwIzeAR6m3aOVltAsj1kfQaml --destination /tmp
```

### Options

```
      --bundle string        The hash id for the bundle, if not specified the latest bundle will be used
      --destination string   The path to the download dir. Defaults to some random dir /tmp/datamon-mount-destination{xxxxx}
      --file string          The file to download from the bundle
  -h, --help                 help for file
      --label string         The human-readable name of a label
      --repo string          The name of this repository
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

* [datamon bundle download](datamon_bundle_download.md)	 - Download a bundle

