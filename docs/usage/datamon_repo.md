**Version: dev**

## datamon repo

Commands to manage repos

### Synopsis

Commands to manage repos.

A datamon repository is analogous to a git repository.

Repos are datasets with a unified lifecycle.
They are versioned and managed via bundles.


### Options

```
  -h, --help   help for repo
```

### Options inherited from parent commands

```
      --config string     Set the config backend store to use (bucket name: do not set the scheme, e.g. 'gs://')
      --context string    Set the context for datamon (default "dev")
      --loglevel string   The logging level. Levels by increasing order of verbosity: none, error, warn, info, debug (default "info")
      --upgrade           Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon](datamon.md)	 - Datamon helps build ML pipelines
* [datamon repo create](datamon_repo_create.md)	 - Create a named repo
* [datamon repo get](datamon_repo_get.md)	 - Get repo info by name
* [datamon repo list](datamon_repo_list.md)	 - List repos

