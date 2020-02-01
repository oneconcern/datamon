**Version: dev**

## datamon label set

Set labels

### Synopsis

Set the label corresponding to a bundle.

Setting a label is analogous to the git command "git tag {label}".

```
datamon label set [flags]
```

### Examples

```
% datamon label set --repo ritesh-test-repo --label anotherlabel --bundle 1ISwIzeAR6m3aOVltAsj1kfQaml

```

### Options

```
      --bundle string   The hash id for the bundle, if not specified the latest bundle will be used
  -h, --help            help for set
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

* [datamon label](datamon_label.md)	 - Commands to manage labels for a repo

