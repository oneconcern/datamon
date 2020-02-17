**Version: dev**

## datamon diamond

Commands to manage diamonds

### Synopsis

A diamond is a parallel data upload operation, which ends up in a single bundle commit.

### Options

```
      --format string   Pretty-print datamon objects using a Go template. Use '{{ printf "%#v" . }}' to explore available fields
  -h, --help            help for diamond
```

### Options inherited from parent commands

```
      --config string     Set the config backend store to use (bucket name: do not set the scheme, e.g. 'gs://')
      --context string    Set the context for datamon (default "dev")
      --loglevel string   The logging level. Levels by increasing order of verbosity: none, error, warn, info, debug (default "info")
      --upgrade           Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon](datamon.md)	 - Datamon helps build ML pipelines
* [datamon diamond cancel](datamon_diamond_cancel.md)	 - Cancels a diamond
* [datamon diamond commit](datamon_diamond_commit.md)	 - Commits a diamond
* [datamon diamond get](datamon_diamond_get.md)	 - Gets diamond info
* [datamon diamond initialize](datamon_diamond_initialize.md)	 - Starts a new diamond
* [datamon diamond list](datamon_diamond_list.md)	 - Lists diamonds in a repo
* [datamon diamond split](datamon_diamond_split.md)	 - Commands to manage splits

