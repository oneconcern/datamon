**Version: dev**

## datamon label

Commands to manage labels for a repo

### Synopsis

Commands to manage labels for a repo.

A label is a name given to a bundle, analogous to a tag in git.

Labels are a mapping type from human-readable strings to commit hashes.

There's one such map per repo, so in particular, setting a label or uploading a bundle
with a label that already exists overwrites the commit hash previously associated with the
label:  There can be at most one commit hash associated with a label.  Conversely,
multiple labels can refer to the same bundle via its commit hash (bundle ID).

### Examples

```
Latest
production
```

### Options

```
      --context string   Set the context for datamon (default "dev")
  -h, --help             help for label
```

### Options inherited from parent commands

```
      --upgrade   Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon](datamon.md)	 - Datamon helps building ML pipelines
* [datamon label get](datamon_label_get.md)	 - Get bundle info by label
* [datamon label list](datamon_label_list.md)	 - List labels
* [datamon label set](datamon_label_set.md)	 - Set labels
