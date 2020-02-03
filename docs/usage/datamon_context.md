**Version: dev**

## datamon context

Commands to manage contexts.

### Synopsis

Commands to manage contexts. A context is an instance of Datamon with set of repos, runs, labels etc.

### Options

```
  -h, --help   help for context
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
* [datamon context create](datamon_context_create.md)	 - Create a context
* [datamon context get](datamon_context_get.md)	 - Get a context info
* [datamon context list](datamon_context_list.md)	 - List available contexts

