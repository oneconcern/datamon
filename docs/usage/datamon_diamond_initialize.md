**Version: dev**

## datamon diamond initialize

Starts a new diamond

### Synopsis

A new diamond is started and its unique ID returned. Use the diamond ID to start splits within that diamond.

Example:
datamon diamond initialize --repo my-repo
304102BC687E087CC3A811F21D113CCF


```
datamon diamond initialize [flags]
```

### Options

```
  -h, --help          help for initialize
      --repo string   The name of this repository
```

### Options inherited from parent commands

```
      --config string     Set the config backend store to use (bucket name: do not set the scheme, e.g. 'gs://')
      --context string    Set the context for datamon (default "dev")
      --format string     Pretty-print datamon objects using a Go template. Use '{{ printf "%#v" . }}' to explore available fields
      --loglevel string   The logging level. Levels by increasing order of verbosity: none, error, warn, info, debug (default "info")
      --upgrade           Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon diamond](datamon_diamond.md)	 - Commands to manage diamonds

