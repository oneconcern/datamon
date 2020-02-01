**Version: dev**

## datamon repo list

List repos

### Synopsis

List repos that have been created

```
datamon repo list [flags]
```

### Examples

```
% datamon repo list --context ctx2
fred , test fred , Frédéric Bidon , frederic@oneconcern.com , 2019-12-05 14:01:18.181535 +0100 CET
```

### Options

```
      --batch-size int           Number of bundles streamed together as a batch. This can be tuned for performance based on network connectivity (default 1024)
      --concurrency-factor int   Heuristic on the amount of concurrency used by core operations. Concurrent retrieval of metadata is capped by the 'batch-size' parameter. Turn this value down to use less memory, increase for faster operations. (default 500)
  -h, --help                     help for list
```

### Options inherited from parent commands

```
      --config string     Set the config backend store to use (bucket name: do not set the scheme, e.g. 'gs://')
      --context string    Set the context for datamon (default "dev")
      --loglevel string   The logging level. Levels by increasing order of verbosity: none, error, warn, info, debug (default "info")
      --upgrade           Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon repo](datamon_repo.md)	 - Commands to manage repos

