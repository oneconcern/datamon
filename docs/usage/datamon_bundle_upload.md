**Version: dev**

## datamon bundle upload

Upload a bundle

### Synopsis

Upload a bundle consisting of all files stored in a directory,
to the cloud backend storage.

This is analogous to the "git commit" command. A message and a label may be set.


```
datamon bundle upload [flags]
```

### Examples

```
% datamon bundle upload --path /path/to/data/folder --message "The initial commit for the repo" --repo ritesh-test-repo --label init
Uploading blob:0871e8f83bdefd710a7710de14decef2254ffed94ee537d72eef671fa82d72d10015b3758b0a8960c93899af265191b0108663c95ece8377bf89e741e14f2a53, bytes:1440
Uploaded bundle id:1INzQ5TV4vAAfU2PbRFgPfnzEwR
set label 'init'

```

### Options

```
      --concurrency-factor int   Heuristic on the amount of concurrency used by various operations.  Turn this value down to use less memory, increase for faster operations. (default 100)
      --files string             Text file containing list of files separated by newline.
  -h, --help                     help for upload
      --label string             The human-readable name of a label
      --message string           The message describing the new bundle
      --path string              The path to the folder or bucket (gs://<bucket>) for the data
      --repo string              The name of this repository
      --skip-on-error            Skip files encounter errors while reading.The list of files is either generated or passed in. During upload files can be deleted or encounter an error. Setting this flag will skip those files. Default to false
```

### Options inherited from parent commands

```
      --config string     Set the config backend store to use (bucket name: do not set the scheme, e.g. 'gs://')
      --context string    Set the context for datamon (default "dev")
      --format string     Pretty-print datamon objects using a Go template. Use '{{ printf "%#v" . }}' to explore available fields
      --loglevel string   The logging level. Levels by increasing order of verbosity: none, error, warn, info, debug (default "info")
      --upgrade           Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon bundle](datamon_bundle.md)	 - Commands to manage bundles for a repo

