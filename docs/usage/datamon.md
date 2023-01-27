**Version: dev**

## datamon

Datamon helps build ML pipelines

### Synopsis

Datamon helps build ML pipelines by adding versioning, auditing and lineage tracking to cloud storage tools
(e.g. Google GCS, AWS S3).

This is not a replacement for these tools, but rather a way to manage their inputs and outputs.

Datamon works by providing a git like interface to manage data efficiently:
your data buckets are organized in repositories of versioned and tagged bundles of files.


### Options

```
      --config string             Set the config backend store to use (bucket name: do not set the scheme, e.g. 'gs://')
      --context string            Set the context for datamon (default "dev")
      --force                     Forces upgrade even if the current version is not a released version
  -h, --help                      help for datamon
      --loglevel string           The logging level. Levels by increasing order of verbosity: none, error, warn, info, debug (default "info")
      --metrics                   Toggle telemetry and metrics collection
      --metrics-password string   Password to connect to the metrics collector backend. Overrides any password set in URL
      --metrics-url string        Fully qualified URL to an influxdb metrics collector, with optional user and password
      --metrics-user string       User to connect to the metrics collector backend. Overrides any user set in URL
      --upgrade                   Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon bundle](datamon_bundle.md)	 - Commands to manage bundles for a repo
* [datamon config](datamon_config.md)	 - Commands to manage the config file
* [datamon context](datamon_context.md)	 - Commands to manage contexts.
* [datamon diamond](datamon_diamond.md)	 - Commands to manage diamonds
* [datamon label](datamon_label.md)	 - Commands to manage labels for a repo
* [datamon purge](datamon_purge.md)	 - Commands to purge unused blob storage
* [datamon repo](datamon_repo.md)	 - Commands to manage repos
* [datamon upgrade](datamon_upgrade.md)	 - Upgrades datamon to the latest release
* [datamon usage](datamon_usage.md)	 - Generates documentation
* [datamon version](datamon_version.md)	 - prints the version of datamon
* [datamon web](datamon_web.md)	 - Webserver

