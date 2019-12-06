**Version: dev**

## datamon upgrade

Upgrades datamon to the latest release

### Synopsis

Checks for the latest release on github repo then upgrades. By default upgrade is skipped if the current datamon is not a released version

```
datamon upgrade [flags]
```

### Options

```
      --check-version   Checks if a new version is available but does not upgrade
      --force           Forces upgrade even if the current version is not a released version
  -h, --help            help for upgrade
```

### Options inherited from parent commands

```
      --upgrade   Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon](datamon.md)	 - Datamon helps building ML pipelines

