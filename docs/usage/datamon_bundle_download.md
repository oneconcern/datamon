**Version: dev**

## datamon bundle download

Download a bundle

### Synopsis

Download a read-only, non-interactive view of the entire data
that is part of a bundle.

If --bundle is not specified, the latest bundle (aka "commit") will be downloaded.

This is analogous to the git command "git checkout {commit-ish}".

```
datamon bundle download [flags]
```

### Examples

```
# Download a bundle by hash
% datamon bundle download --repo ritesh-test-repo --destination /path/to/folder/to/download --bundle 1INzQ5TV4vAAfU2PbRFgPfnzEwR

# Download a bundle by label
% datamon bundle download --repo ritesh-test-repo --destination /path/to/folder/to/download --label init
Using bundle: 1UZ6kpHe3EBoZUTkKPHSf8s2beh
...

```

### Options

```
      --bundle string            The hash id for the bundle, if not specified the latest bundle will be used
      --concurrency-factor int   Heuristic on the amount of concurrency used by various operations.  Turn this value down to use less memory, increase for faster operations. (default 100)
      --destination string       The path to the download dir. Defaults to some random dir /tmp/datamon-mount-destination{xxxxx}
      --force-dest               Override destination path is empty check
  -h, --help                     help for download
      --label string             The human-readable name of a label
      --name-filter string       A regular expression (RE2) to match names of bundle entries.
      --repo string              The name of this repository
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

* [datamon bundle](datamon_bundle.md)	 - Commands to manage bundles for a repo
* [datamon bundle download file](datamon_bundle_download_file.md)	 - Download a file from bundle

