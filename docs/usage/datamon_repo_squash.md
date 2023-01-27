**Version: dev**

## datamon repo squash

Squash the history of a repo

### Synopsis

Squash a repo so that only the latest bundle remains.

Optionally, the squashing may also retain past tagged bundles, or only past tagged bundles with a legit semver tag.


```
datamon repo squash [flags]
```

### Examples

```
% datamon repo squash  --retain-semver-tags --repo ritesh-datamon-test-repo
```

### Options

```
      --batch-size int           Number of bundles streamed together as a batch. This can be tuned for performance based on network connectivity (default 1024)
      --concurrency-factor int   Heuristic on the amount of concurrency used by core operations. Concurrent retrieval of metadata is capped by the 'batch-size' parameter. Turn this value down to use less memory, increase for faster operations. (default 500)
  -h, --help                     help for squash
      --repo (*) string          The name of this repository
      --retain-n-latest int      Squash past bundles and retain n latest versions. May be combined with retain-tags and retain-semver-flags (default 1)
      --retain-semver-tags       Squash past bundles and retain all semver tagged past bundles
      --retain-tags              Squash past bundles and retain all tagged past bundles
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

* [datamon repo](datamon_repo.md)	 - Commands to manage repos

