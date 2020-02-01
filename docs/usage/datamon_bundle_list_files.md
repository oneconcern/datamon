**Version: dev**

## datamon bundle list files

List files in a bundle

### Synopsis

List all the files in a bundle.

You may use the "--label" flag as an alternate way to specify the bundle to search for.

This is analogous to the git command "git show --pretty="" --name-only {commit-ish}".


```
datamon bundle list files [flags]
```

### Examples

```
% datamon bundle list files --repo ritesh-test-repo --bundle 1ISwIzeAR6m3aOVltAsj1kfQaml
Using bundle: 1UZ6kpHe3EBoZUTkKPHSf8s2beh
name:bundle_upload.go, size:4021, hash:b9258e91eb29fe42c70262dd2da46dd71385995dbb989e6091328e6be3d9e3161ad22d9ad0fbfb71410f9e4730f6ac4482cc592c0bc6011585bd9b0f00b11463
...
```

### Options

```
      --bundle string   The hash id for the bundle, if not specified the latest bundle will be used
  -h, --help            help for files
      --label string    The human-readable name of a label
      --repo string     The name of this repository
```

### Options inherited from parent commands

```
      --config string     Set the config backend store to use (bucket name: do not set the scheme, e.g. 'gs://')
      --context string    Set the context for datamon (default "dev")
      --loglevel string   The logging level. Levels by increasing order of verbosity: none, error, warn, info, debug (default "info")
      --upgrade           Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon bundle list](datamon_bundle_list.md)	 - List bundles

