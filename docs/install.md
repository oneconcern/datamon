# Installation

## From docker

## From binary release

Download the datamon binary for mac or for linux on the
[Releases Page](https://github.com/oneconcern/datamon/releases/)
or use the
[shell wrapper](#os-x-install-guide)

Example:
```$bash
tar -zxvf datamon.mac.tgz
```
## From source

```bash
go get -u github.com/oneconcern/datamon
```

## OS X install guide

The recommended way to install datamon in your local environemnt is to use the
`deploy/datamon.sh` wrapper script.  This script is responsible for downloading
the datamon binary from the [Releases Page](https://github.com/oneconcern/datamon/releases/),
keeping a local cache of binaries, and `exec`ing the binary.  So parameterization
of the shell script is the same as parameterization as the binary:  the shell script
is transparent.

Download the script, set it to be executable, and then try the `version` verb in the
wrapped binary to verify that the binary is installed locally.  There are several
auxilliary programs required by the shell script such as `grep` and `wget`.  If these
are not installed, the script will exit with a descriptive message, and the missing
utility can be installed via [`brew`](https://docs.brew.sh/) or otherwise.

```
curl https://raw.githubusercontent.com/oneconcern/datamon/master/deploy/datamon.sh -o datamon
chmod +x datamon
./datamon version
```

It's probably most convenient to have the wrapper script placed somewhere on your
shell's path, of course.

