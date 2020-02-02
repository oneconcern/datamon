**Version: dev**

## datamon label get

Get bundle info by label

### Synopsis

Performs a direct lookup of labels by name.
Prints corresponding bundle information if the label exists,
exits with ENOENT status otherwise.

```
datamon label get [flags]
```

### Options

```
  -h, --help           help for get
      --label string   The human-readable name of a label
      --repo string    The name of this repository
```

### Options inherited from parent commands

```
      --config string     Set the config backend store to use (bucket name: do not set the scheme, e.g. 'gs://')
      --context string    Set the context for datamon (default "dev")
      --loglevel string   The logging level. Levels by increasing order of verbosity: none, error, warn, info, debug (default "info")
      --upgrade           Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon label](datamon_label.md)	 - Commands to manage labels for a repo

