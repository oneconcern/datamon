**Version: dev**

## datamon bundle

Commands to manage bundles for a repo

### Synopsis

Commands to manage bundles for a repo.

A bundle is a point in time read-only view of a repo,
analogous to a git commit.

A bundle is composed of individual files that are tracked and changed
together.

### Options

```
      --format string   Pretty-print datamon objects using a Go template. Use '{{ printf "%#v" . }}' to explore available fields
  -h, --help            help for bundle
```

### Options inherited from parent commands

```
      --config string     Set the config backend store to use (bucket name: do not set the scheme, e.g. 'gs://')
      --context string    Set the context for datamon (default "dev")
      --loglevel string   The logging level. Levels by increasing order of verbosity: none, error, warn, info, debug (default "info")
      --upgrade           Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon](datamon.md)	 - Datamon helps building ML pipelines
* [datamon bundle diff](datamon_bundle_diff.md)	 - Diff a downloaded bundle with a remote bundle.
* [datamon bundle download](datamon_bundle_download.md)	 - Download a bundle
* [datamon bundle get](datamon_bundle_get.md)	 - Get bundle info
* [datamon bundle list](datamon_bundle_list.md)	 - List bundles
* [datamon bundle mount](datamon_bundle_mount.md)	 - Mount a bundle
* [datamon bundle update](datamon_bundle_update.md)	 - Update a downloaded bundle with a remote bundle.
* [datamon bundle upload](datamon_bundle_upload.md)	 - Upload a bundle

