# Installation

## From docker

Docker releases are available from Oneconcern Google private registry only:

```
docker pull gcr.io/onec-co/datamon:v2.1.0
```

Other released images:
```
gcr.io/onec-co/datamon-fuse-sidecar     # a fuse-mount sidecar to mount a datamon bundle on a pod
gcr.io/onec-co/datamon-pg-sidecar       # a postgres-enabled sidecar to spin up databases from a datamon bundle
gcr.io/onec-co/datamon-wrapper          # a wrapper script to use with datamon sidecars
gcr.io/onec-co/migrate                  # a standalone data copy tool
```

Detail about the contents of these images are available [here](../dockerfiles/README.md)

## From binary release

Download the datamon binary for mac or for linux on the
[Releases Page](https://github.com/oneconcern/datamon/releases/)

To get version 1.0:
Unzip the tar.gz file and run the following command to move the executable to the correct destination:
```
mv ~/Downloads/datamon /usr/local/bin/
```

If you run ```datamon version```, you should see ```Version: v1.0.0```.

Example:
```$bash
download_url=$(curl -s https://api.github.com/repos/oneconcern/datamon/releases/latest | \
  jq -r '.assets[] | select(.name | contains("'"$(uname | tr '[:upper:]' '[:lower:]')"'_amd64")) | .browser_download_url')
curl -o /usr/local/bin/datamon -L'#' "$download_url"
chmod +x /usr/local/bin/datamon
```

## From source

```bash
go install github.com/oneconcern/datamon@latest
```

## Homebrew/Linuxbrew

```
brew tap oneconcern/datamon
brew install datamon
```
