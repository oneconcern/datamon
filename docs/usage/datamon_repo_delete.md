**Version: dev**

## datamon repo delete

Delete a named repo

### Synopsis

Delete an existing datamon repository.

You must authenticate to perform this operation (can't --skip-auth).
You must specify the context with --context.

This command MUST NOT BE RUN concurrently.


```
datamon repo delete [flags]
```

### Examples

```
% datamon repo delete --repo ritesh-datamon-test-repo --context dev
```

### Options

```
  -h, --help              help for delete
      --repo (*) string   The name of this repository
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

* [datamon repo](datamon_repo.md)	 - Commands to manage repos
* [datamon repo delete files](datamon_repo_delete_files.md)	 - Deletes files from a named repo, altering all bundles

