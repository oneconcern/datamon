**Version: dev**

## datamon version

prints the version of datamon

### Synopsis

Prints the version of datamon. It includes the following components:
	* Semver (output of git describe --tags)
	* Build Date (date at which the binary was built)
	* Git Commit (the git commit hash this binary was built from
	* Git State (when dirty there were uncommitted changes during the build)


```
datamon version [flags]
```

### Options

```
      --format string   Pretty-print datamon objects using a Go template. Use '{{ printf "%#v" . }}' to explore available fields
  -h, --help            help for version
```

### Options inherited from parent commands

```
      --config string     Set the config backend store to use (bucket name: do not set the scheme, e.g. 'gs://')
      --context string    Set the context for datamon (default "dev")
      --loglevel string   The logging level. Levels by increasing order of verbosity: none, error, warn, info, debug (default "info")
      --upgrade           Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon](datamon.md)	 - Datamon helps building ML pipelines

