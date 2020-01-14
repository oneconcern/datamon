**Version: dev**

## datamon bundle mount new

Create a bundle incrementally with filesystem operations

### Synopsis

Write directories and files to the mountpoint.  Unmount or send SIGINT to this process to save.
The destination path is a temporary staging area for write operations.

```
datamon bundle mount new [flags]
```

### Options

```
      --daemonize            Whether to run the command as a daemonized process
      --destination string   The path to the download dir. Defaults to some random dir /tmp/datamon-mount-destination{xxxxx}
  -h, --help                 help for new
      --label string         The human-readable name of a label
      --loglevel string      The logging level (default "info")
      --message string       The message describing the new bundle
      --mount string         The path to the mount dir
      --repo string          The name of this repository
```

### Options inherited from parent commands

```
      --config string    Set the config backend store to use (bucket name: do not set the scheme, e.g. 'gs://')
      --context string   Set the context for datamon (default "dev")
      --upgrade          Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon bundle mount](datamon_bundle_mount.md)	 - Mount a bundle

