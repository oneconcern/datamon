**Version: dev**

## datamon bundle mount

Mount a bundle

### Synopsis

Mount a readonly, non-interactive view of the entire data that is part of a bundle

```
datamon bundle mount [flags]
```

### Options

```
      --bundle string            The hash id for the bundle, if not specified the latest bundle will be used
      --concurrency-factor int   Heuristic on the amount of concurrency used by various operations.  Turn this value down to use less memory, increase for faster operations. (default 100)
      --cpuprof                  Toggle runtime profiling
      --daemonize                Whether to run the command as a daemonized process
      --destination string       The path to the download dir
  -h, --help                     help for mount
      --label string             The human-readable name of a label
      --loglevel string          The logging level (default "info")
      --mount string             The path to the mount dir
      --repo string              The name of this repository
      --stream                   Stream in the FS view of the bundle, do not download all files. Default to true. (default true)
```

### Options inherited from parent commands

```
      --context string   Set the context for datamon (default "dev")
      --upgrade          Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon bundle](datamon_bundle.md)	 - Commands to manage bundles for a repo
* [datamon bundle mount new](datamon_bundle_mount_new.md)	 - Create a bundle incrementally with filesystem operations

