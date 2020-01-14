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
      --cache-size byte-size     The desired size of the memory cache used when streaming is enabled (default 50MB)
      --concurrency-factor int   Heuristic on the amount of concurrency used by various operations.  Turn this value down to use less memory, increase for faster operations. (default 100)
      --cpuprof                  Toggle runtime profiling
      --daemonize                Whether to run the command as a daemonized process
      --destination string       The path to the download dir. Defaults to some random dir /tmp/datamon-mount-destination{xxxxx}
  -h, --help                     help for mount
      --label string             The human-readable name of a label
      --loglevel string          The logging level (default "info")
      --mount string             The path to the mount dir
      --prefetch int             Enables prefetching (requires Stream enabled) (default 1)
      --repo string              The name of this repository
      --stream                   Stream in the FS view of the bundle, do not download all files. Default to true. (default true)
      --verify-hash              Enables hash verification on read blobs (requires Stream enabled) (default true)
```

### Options inherited from parent commands

```
      --config string    Set the config backend store to use (bucket name: do not set the scheme, e.g. 'gs://')
      --context string   Set the context for datamon (default "dev")
      --upgrade          Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon bundle](datamon_bundle.md)	 - Commands to manage bundles for a repo
* [datamon bundle mount new](datamon_bundle_mount_new.md)	 - Create a bundle incrementally with filesystem operations

