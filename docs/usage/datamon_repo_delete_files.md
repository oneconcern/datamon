**Version: dev**

## datamon repo delete files

Deletes files from a named repo, altering all bundles

### Synopsis

Deletes files in a file list from all bundles in an existing datamon repository.

This command MUST NOT BE RUN concurrently.


```
datamon repo delete files [flags]
```

### Examples

```

% datamon repo delete files --repo ritesh-datamon-test-repo --files file-list.txt

% datamon repo delete files --repo ritesh-datamon-test-repo --file path/file-to-delete

```

### Options

```
      --file string    The file to download from the bundle
      --files string   Text file containing list of files separated by newline.
  -h, --help           help for files
      --repo string    The name of this repository
```

### Options inherited from parent commands

```
      --config string             Set the config backend store to use (bucket name: do not set the scheme, e.g. 'gs://')
      --context string            Set the context for datamon (default "dev")
      --format string             Pretty-print datamon objects using a Go template. Use '{{ printf "%#v" . }}' to explore available fields
      --loglevel string           The logging level. Levels by increasing order of verbosity: none, error, warn, info, debug (default "info")
      --metrics                   Toggle telemetry and metrics collection
      --metrics-password string   Password to connect to the metrics collector backend. Overrides any password set in URL
      --metrics-url string        Fully qualified URL to an influxdb metrics collector, with optional user and password
      --metrics-user string       User to connect to the metrics collector backend. Overrides any user set in URL
      --upgrade                   Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon repo delete](datamon_repo_delete.md)	 - Delete a named repo

