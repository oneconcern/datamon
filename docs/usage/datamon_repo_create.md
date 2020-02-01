**Version: dev**

## datamon repo create

Create a named repo

### Synopsis

Creates a new datamon repository.

Repo names must not contain special characters.
Allowed characters Unicode characters, digits and hyphen.

This is analogous to the "git init ..." command.

```
datamon repo create [flags]
```

### Examples

```
% datamon repo create  --description "Ritesh's repo for testing" --repo ritesh-datamon-test-repo
```

### Options

```
      --description string   The description for the repo
  -h, --help                 help for create
      --repo string          The name of this repository
```

### Options inherited from parent commands

```
<<<<<<< HEAD
      --config string     Set the config backend store to use (bucket name: do not set the scheme, e.g. 'gs://')
      --context string    Set the context for datamon (default "dev")
      --loglevel string   The logging level. Levels by increasing order of verbosity: none, error, warn, info, debug (default "info")
      --upgrade           Upgrades the current version then carries on with the specified command
=======
      --config string    Set the config backend store to use (bucket name: do not set the scheme, e.g. 'gs://')
      --context string   Set the context for datamon (default "dev")
      --format string    Pretty-print datamon objects using a Go template. Use '{{ printf "%#v" . }}' to explore available fields
      --upgrade          Upgrades the current version then carries on with the specified command
>>>>>>> feat(cli): custom format templating for output
```

### SEE ALSO

* [datamon repo](datamon_repo.md)	 - Commands to manage repos

