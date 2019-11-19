# Usage

## Configure datamon

For non kubernetes use, it's necessary to supply gcloud credentials.

```bash
# Replace path to gcloud credential file. Use absolute path
% datamon config create --credential /Users/ritesh/.config/gcloud/application_default_credentials.json
```

Inside a kubernetes pod, Datamon will use kubernetes service credentials.

Check the config file, credential file will not be set in kubernetes deployment.
```bash
% cat ~/.datamon/datamon.yaml
metadata: datamon-meta-data
blob: datamon-blob-data
email: ritesh@oneconcern.com
name: Ritesh H Shukla
credential: /Users/ritesh/.config/gcloud/application_default_credentials.json
```
## Authentication

Datamon keeps track of who contributed what. The identity of contributors
is obtained from an OIDC identity provider (Google ID). 

Make sure your gcloud credentials have been setup, with proper scopes.
```$bash
gcloud auth application-default login --scopes https://www.googleapis.com/auth/cloud-platform,email,profile
```

datamon will use your email and name from your Google ID account.

> **NOTE**: by default `gcloud auth application-default login` will not allow applications to see your full profile
> In that case, datamon will use your email as your user name.
>
> You may control your personal information stored by Google here: https://aboutme.google.com

## Create repo

Datamon repos are analogous to git repos.

```bash
% datamon repo create  --description "Ritesh's repo for testing" --repo ritesh-datamon-test-repo
```

## Upload a bundle

The last line prints the commit hash.
If the optional `--label` is omitted, the commit hash will be needed to download the bundle.
```bash
% datamon bundle upload --path /path/to/data/folder --message "The initial commit for the repo" --repo ritesh-test-repo --label init
Uploaded bundle id:1INzQ5TV4vAAfU2PbRFgPfnzEwR
```

## List bundles
List all the bundles in a particular repo.
```bash
% datamon bundle list --repo ritesh-test-repo
Using config file: /Users/ritesh/.datamon/datamon.yaml
1INzQ5TV4vAAfU2PbRFgPfnzEwR , 2019-03-12 22:10:24.159704 -0700 PDT , Updating test bundle
```

## List labels
List all the labels in a particular repo.
```bash
% datamon label list --repo ritesh-test-repo
Using config file: /Users/ritesh/.datamon/datamon.yaml
init , 1INzQ5TV4vAAfU2PbRFgPfnzEwR , 2019-03-12 22:10:24.159704 -0700 PDT
```

## Download a bundle

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
```bash
datamon bundle list files --repo ritesh-test-repo --bundle 1ISwIzeAR6m3aOVltAsj1kfQaml
```

Also uses `--label` flag as an alternate way to specify the bundle in question.

## Download a file
Download a single file from a bundle
```bash
datamon bundle download file --file datamon/cmd/repo_list.go --repo ritesh-test-repo --bundle 1ISwIzeAR6m3aOVltAsj1kfQaml --destination /tmp
```

Can also use the `--label` as an alternate way to specify the particular bundle.

## Set a label

```bash
% datamon label set --repo ritesh-test-repo --label anotherlabel --bundle 1ISwIzeAR6m3aOVltAsj1kfQaml
Uploaded bundle id:1INzQ5TV4vAAfU2PbRFgPfnzEwR
```

Labels are a mapping type from human-readable strings to commit hashes.

There's one such map per repo, so in particular setting a label or uploading a bundle
with a label that already exists overwrites the commit hash previously associated with the
label:  There can be at most one commit hash associated with a label.  Conversely,
multiple labels can refer to the same bundle via its commit hash.
