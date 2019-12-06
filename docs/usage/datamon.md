**Version: dev**

## datamon

Datamon helps building ML pipelines

### Synopsis

Datamon helps building ML pipelines by adding versioning, auditing and security to cloud storage tools
(e.g. Google GCS, AWS S3).

This is not a replacement for these tools, but rather a way to manage their inputs and outputs.

Datamon works by providing a git like interface to manage data efficiently:
your data buckets are organized in repositories of versioned and tagged bundles of files.


### Options

```
      --config string   Set the config backend store to use (do not set the scheme, e.g. 'gs://')
      --force           Forces upgrade even if the current version is not a released version
  -h, --help            help for datamon
      --upgrade         Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon bundle](datamon_bundle.md)	 - Commands to manage bundles for a repo
* [datamon config](datamon_config.md)	 - Commands to manage a config
* [datamon context](datamon_context.md)	 - Commands to manage contexts.
* [datamon label](datamon_label.md)	 - Commands to manage labels for a repo
* [datamon repo](datamon_repo.md)	 - Commands to manage repos
* [datamon upgrade](datamon_upgrade.md)	 - Upgrades datamon to the latest release
* [datamon usage](datamon_usage.md)	 - Generates documentation
* [datamon version](datamon_version.md)	 - prints the version of datamon
* [datamon web](datamon_web.md)	 - Webserver

