**Version: dev**

## datamon diamond split add

adds a new split and starts uploading

### Synopsis

adds a new split and starts uploading

```
datamon diamond split add [flags]
```

### Options

```
      --concurrency-factor int   Heuristic on the amount of concurrency used by various operations.  Turn this value down to use less memory, increase for faster operations. (default 100)
      --diamond string           The diamond to use
      --files string             Text file containing list of files separated by newline.
  -h, --help                     help for add
      --name-filter string       A regular expression (RE2) to match names of bundle entries.
      --path string              The path to the folder or bucket (gs://<bucket>) for the data
      --repo string              The name of this repository
      --skip-on-error            Skip files encounter errors while reading.The list of files is either generated or passed in. During upload files can be deleted or encounter an error. Setting this flag will skip those files. Default to false
      --split string             The split to use
      --split-tag string         A custom tag to identify your split in logs or datamon reports. Example: "pod-1"
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

* [datamon diamond split](datamon_diamond_split.md)	 - Commands to manage splits

