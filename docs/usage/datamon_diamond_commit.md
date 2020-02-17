**Version: dev**

## datamon diamond commit

Commits a diamond

### Synopsis

Commits a diamond to create a bundle, with conflicts handling

```
datamon diamond commit [flags]
```

### Options

```
      --concurrency-factor int   Heuristic on the amount of concurrency used by various operations.  Turn this value down to use less memory, increase for faster operations. (default 100)
      --diamond string           The diamond to use
      --diamond-tag string       A custom tag to identify your diamond in logs or datamon reports. Example: "coordinator-pod-A"
  -h, --help                     help for commit
      --ignore-conflicts         Diamond commit ignores conflicts and does not report about any
      --label string             The human-readable name of a label
      --message string           The message describing the new bundle
      --no-conflicts             Diamond commit fails if any conflict is detected
      --repo string              The name of this repository
      --with-checkpoints         Diamond commit handles conflicts and keeps them as intermediate checkpoints rather than conflicts
      --with-conflicts           Diamond commit handles conflicts and keeps them in store (default true)
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

