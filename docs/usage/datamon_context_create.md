**Version: dev**

## datamon context create

Create a context

### Synopsis

Create a context for Datamon

```
datamon context create [flags]
```

### Options

```
      --blob string       The name of the bucket hosting the datamon blobs
      --config string     Set the config backend store to use
      --context string    Set the context for datamon (default "dev")
  -h, --help              help for create
      --meta string       The name of the bucket used by datamon metadata
      --read-log string   The name of the bucket hosting the read log
      --vmeta string      The name of the bucket hosting the versioned metadata
      --wal string        The name of the bucket hosting the WAL
```

### Options inherited from parent commands

```
      --upgrade   Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon context](datamon_context.md)	 - Commands to manage contexts.

