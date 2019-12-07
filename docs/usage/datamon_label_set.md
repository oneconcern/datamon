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
      --context string   Set the context for datamon (default "dev")
      --upgrade          Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon label](datamon_label.md)	 - Commands to manage labels for a repo

