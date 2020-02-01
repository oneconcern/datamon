**Version: dev**

## datamon bundle get

Get bundle info

### Synopsis

Performs a direct lookup of a bundle.

Prints corresponding bundle metadata if the bundle exists,
exits with ENOENT status otherwise.

```
datamon bundle get [flags]
```

### Options

```
      --bundle string   The hash id for the bundle, if not specified the latest bundle will be used
  -h, --help            help for get
      --label string    The human-readable name of a label
      --repo string     The name of this repository
```

### Options inherited from parent commands

```
<<<<<<< HEAD
      --config string     Set the config backend store to use (bucket name: do not set the scheme, e.g. 'gs://')
      --context string    Set the context for datamon (default "dev")
      --loglevel string   The logging level. Levels by increasing order of verbosity: none, error, warn, info, debug (default "info")
      --upgrade           Upgrades the current version then carries on with the specified command
=======
      --config string    Set the config backend store to use (bucket name: do not set the scheme, e.g. 'gs://')
      --context string   Set the context for datamon (default "dev")
      --format string    Pretty-print datamon objects using a Go template. Use '{{ printf "%#v" . }}' to explore available fields
      --upgrade          Upgrades the current version then carries on with the specified command
>>>>>>> feat(cli): custom format templating for output
```

### SEE ALSO

* [datamon bundle](datamon_bundle.md)	 - Commands to manage bundles for a repo

