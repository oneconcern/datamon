# Installation

## From docker

Docker releases are available from Oneconcern Google private registry only:

```
docker pull gcr.io/onec-co/datamon
```

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
go get -u github.com/oneconcern/datamon
```

## Homebrew/Linuxbrew

```
brew tap oneconcern/datamon
brew install datamon
```
