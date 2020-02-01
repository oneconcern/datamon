**Version: dev**

## datamon context get

Get a context info

### Synopsis

Get a Datamon context's info

```
datamon context get [flags]
```

### Options

```
  -h, --help   help for get
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

* [datamon context](datamon_context.md)	 - Commands to manage contexts.

