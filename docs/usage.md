# Usage

## CLI

Datamon helps building ML pipelines by adding versioning, auditing and security to existing tools.

This is not a replacement for existing tools, but rather a way to manage their inputs and outputs.

Datamon works by providing a git like interface to manage data efficiently.

```
Usage:
  datamon [command]

Available Commands:
  bundle      Commands to manage bundles for a repo
  config      Commands to manage a config
  context     Commands to manage contexts.
  help        Help about any command
  label       Commands to manage labels for a repo
  repo        Commands to manage repos
  version     prints the version of this datamon
  web         Webserver

Flags:
      --config string   Set the config to use
  -h, --help            help for datamon

Use "datamon [command] --help" for more information about a command.
```

## Configure datamon

```
Commands to manage datamon cli config.

Configuration for datamon is the common set of datamonFlags that are needed for most commands and do not change.

Usage:
  datamon config [command]

Available Commands:
  create      Create a config

Flags:
  -h, --help   help for config
```

> git analogy: `git config ...`

Check the config file (credential file will is not set in kubernetes deployment)

### Authentication

For non kubernetes use, gcloud credentials are forwarded by default.
Inside a kubernetes pod, Datamon will use kubernetes service credentials.

Datamon keeps track of who contributed what. The identity of contributors
is obtained from an OIDC identity provider (Google ID). 

Make sure your gcloud credentials have been setup, with proper scopes.

```bash
gcloud auth application-default login --scopes https://www.googleapis.com/auth/cloud-platform,email,profile
```

```bash
% cat ~/.datamon/datamon.yaml
metadata: datamon-meta-data
blob: datamon-blob-data
credential: /Users/ritesh/.config/gcloud/application_default_credentials.json
```

> NOTE:  this assume the default location for gcloud credential is `~/.config/gcloud/application_default_credentials.json`
> You may be overriden as in the example below.

**Example:**
```bash
# Replace path to gcloud credential file. Use absolute path
% datamon config create --credential /Users/ritesh/.config/gcloud/application_default_credentials.json
```

datamon will use your email and name from your Google ID account.

> **NOTE**: by default `gcloud auth application-default login` will not allow applications to see your full profile
> In that case, datamon will use your email as your user name.
>
> You may control your personal information stored by Google here: https://aboutme.google.com

## Create repo

Datamon repos are analogous to git repos.

```
Create a repo. Repo names must not contain special characters. Allowed characters Unicode characters, digits and hyphen. Example: dm-test-repo-1

Usage:
  datamon repo create [flags]

Flags:
      --description string   The description for the repo
  -h, --help                 help for create
      --repo string          The name of this repository
```

> git analogy: `git init ...`

**Example:**

```bash
% datamon repo create  --description "Ritesh's repo for testing" --repo ritesh-datamon-test-repo
```

## List repos

List repos that have been created.

```
Usage:
  datamon repo list [flags]

Flags:
      --batch-size int           Number of bundles streamed together as a batch. This can be tuned for performance based on network connectivity (default 1024)
      --concurrency-factor int   Heuristic on the amount of concurrency used by core operations. Concurrent retrieval of metadata is capped by the 'batch-size' parameter. Turn this value down to use less memory, increase for faster operations. (default 500)
  -h, --help                     help for list
```

## Upload a bundle

Upload a bundle consisting of all files stored in a directory.

```
Usage:
  datamon bundle upload [flags]

Flags:
      --concurrency-factor int   Heuristic on the amount of concurrency used by various operations.  Turn this value down to use less memory, increase for faster operations. (default 100)
      --files string             Text file containing list of files separated by newline.
  -h, --help                     help for upload
      --label string             The human-readable name of a label
      --loglevel string          The logging level (default "info")
      --message string           The message describing the new bundle
      --path string              The path to the folder or bucket (gs://<bucket>) for the data
      --repo string              The name of this repository
      --skip-on-error            Skip files encounter errors while reading.The list of files is either generated or passed in. During upload files can be deleted or encounter an error. Setting this flag will skip those files. Default to false
```

The last line prints the commit hash.
If the optional `--label` is omitted, the commit hash will be needed to download the bundle.

**Example:**

```bash
% datamon bundle upload --path /path/to/data/folder --message "The initial commit for the repo" --repo ritesh-test-repo --label init
Uploaded bundle id:1INzQ5TV4vAAfU2PbRFgPfnzEwR
```

## List bundles

List the bundles in a repo, ordered by their bundle ID

```
Usage:
  datamon bundle list [flags]
  datamon bundle list [command]

Available Commands:
  files       List files in a bundle

Flags:
      --batch-size int           Number of bundles streamed together as a batch. This can be tuned for performance based on network connectivity (default 1024)
      --concurrency-factor int   Heuristic on the amount of concurrency used by core operations. Concurrent retrieval of metadata is capped by the 'batch-size' parameter. Turn this value down to use less memory, increase for faster operations. (default 500)
  -h, --help                     help for list
      --repo string              The name of this repository
```

> git analogy: `git log`

**Example:**

```bash
% datamon bundle list --repo ritesh-test-repo
Using config file: /Users/ritesh/.datamon/datamon.yaml
1INzQ5TV4vAAfU2PbRFgPfnzEwR , 2019-03-12 22:10:24.159704 -0700 PDT , Updating test bundle
```

## List labels

List all the labels in a particular repo.

```
Usage:
  datamon label list [flags]

Flags:
      --batch-size int           Number of bundles streamed together as a batch. This can be tuned for performance based on network connectivity (default 1024)
      --concurrency-factor int   Heuristic on the amount of concurrency used by core operations. Concurrent retrieval of metadata is capped by the 'batch-size' parameter. Turn this value down to use less memory, increase for faster operations. (default 500)
  -h, --help                     help for list
      --prefix string            List labels starting with a prefix.
      --repo string              The name of this repository
```

> git analogy: `git tag --list`

**Example:**

```bash
% datamon label list --repo ritesh-test-repo
Using config file: /Users/ritesh/.datamon/datamon.yaml
init , 1INzQ5TV4vAAfU2PbRFgPfnzEwR , 2019-03-12 22:10:24.159704 -0700 PDT
```

## Download a bundle

```
Download a readonly, non-interactive view of the entire data that is part of a bundle. If --bundle is not specified the latest bundle will be downloaded

Usage:
  datamon bundle download [flags]
  datamon bundle download [command]

Available Commands:
  file        Download a file from bundle

Flags:
      --bundle string            The hash id for the bundle, if not specified the latest bundle will be used
      --concurrency-factor int   Heuristic on the amount of concurrency used by various operations.  Turn this value down to use less memory, increase for faster operations. (default 100)
      --destination string       The path to the download dir
  -h, --help                     help for download
      --label string             The human-readable name of a label
      --name-filter string       A regular expression (RE2) to match names of bundle entries.
```

> git analogy: `git checkout {commit-ish}`

**Examples:**

Download a bundle by either hash

```bash
datamon bundle download --repo ritesh-test-repo --destination /path/to/folder/to/download --bundle 1INzQ5TV4vAAfU2PbRFgPfnzEwR
```

or label

```bash
datamon bundle download --repo ritesh-test-repo --destination /path/to/folder/to/download --label init
```

## List bundle contents

List all files in a bundle

```
Usage:
  datamon bundle list files [flags]

Flags:
      --bundle string   The hash id for the bundle, if not specified the latest bundle will be used
  -h, --help            help for files
      --label string    The human-readable name of a label
      --repo string     The name of this repository
```

> git analogy: ` git show --pretty="" --name-only {commit-ish}`

**Example:**

```bash
datamon bundle list files --repo ritesh-test-repo --bundle 1ISwIzeAR6m3aOVltAsj1kfQaml
```

Also uses `--label` flag as an alternate way to specify the bundle in question.

## Download a file
Download a readonly, non-interactive view of a single file from a bundle.

```
Usage:
  datamon bundle download file [flags]

Flags:
      --bundle string        The hash id for the bundle, if not specified the latest bundle will be used
      --destination string   The path to the download dir
      --file string          The file to download from the bundle
  -h, --help                 help for file
      --label string         The human-readable name of a label
      --repo string          The name of this repository
```

**Example:**

```bash
datamon bundle download file --file datamon/cmd/repo_list.go --repo ritesh-test-repo --bundle 1ISwIzeAR6m3aOVltAsj1kfQaml --destination /tmp
```

Can also use the `--label` as an alternate way to specify the particular bundle.

## Set a label

Set the label corresponding to a bundle.

Labels are a mapping type from human-readable strings to commit hashes.

There's one such map per repo, so in particular, setting a label or uploading a bundle
with a label that already exists overwrites the commit hash previously associated with the
label:  There can be at most one commit hash associated with a label.  Conversely,
multiple labels can refer to the same bundle via its commit hash.

```
Usage:
  datamon label set [flags]

Flags:
      --bundle string   The hash id for the bundle, if not specified the latest bundle will be used
  -h, --help            help for set
      --label string    The human-readable name of a label
      --repo string     The name of this repository
```

> git analogy: `git tag {label}`

**Example:**

```bash
% datamon label set --repo ritesh-test-repo --label anotherlabel --bundle 1ISwIzeAR6m3aOVltAsj1kfQaml
Uploaded bundle id:1INzQ5TV4vAAfU2PbRFgPfnzEwR
```
