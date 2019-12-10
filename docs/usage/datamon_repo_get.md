**Version: dev**

## datamon repo get

Get repo info by name

### Synopsis

Performs a direct lookup of repos by name.
Prints corresponding repo information if the name exists,
exits with ENOENT status otherwise.

```
datamon repo get [flags]
```

### Options

```
  -h, --help          help for get
      --repo string   The name of this repository
```

### Options inherited from parent commands

```
      --context string   Set the context for datamon (default "dev")
      --upgrade          Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon repo](datamon_repo.md)	 - Commands to manage repos
