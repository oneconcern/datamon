**Version: dev**

## datamon bundle download

Download a bundle

### Synopsis

Download a read-only, non-interactive view of the entire data
that is part of a bundle.

If --bundle is not specified, the latest bundle (aka "commit") will be downloaded.

This is analogous to the git command "git checkout {commit-ish}".

```
datamon bundle download [flags]
```

### Examples

```
# Download a bundle by hash
% datamon bundle download --repo ritesh-test-repo --destination /path/to/folder/to/download --bundle 1INzQ5TV4vAAfU2PbRFgPfnzEwR

# Download a bundle by label
% datamon bundle download --repo ritesh-test-repo --destination /path/to/folder/to/download --label init
Using bundle: 1UZ6kpHe3EBoZUTkKPHSf8s2beh
...

```

### Options

```
      --bundle string            The hash id for the bundle, if not specified the latest bundle will be used
      --concurrency-factor int   Heuristic on the amount of concurrency used by various operations.  Turn this value down to use less memory, increase for faster operations. (default 100)
      --destination string       The path to the download dir
  -h, --help                     help for download
      --label string             The human-readable name of a label
      --name-filter string       A regular expression (RE2) to match names of bundle entries.
      --repo string              The name of this repository
```

### Options inherited from parent commands

```
      --context string   Set the context for datamon (default "dev")
      --upgrade          Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon bundle](datamon_bundle.md)	 - Commands to manage bundles for a repo
* [datamon bundle download file](datamon_bundle_download_file.md)	 - Download a file from bundle

