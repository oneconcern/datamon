# Datamover container guide

As with the [Kubernetes sidecar guide](#kubernetes-sidecar-guide), this section covers
a particular operationalization of Datamon at One Concern wherein we use the program
along with some auxilliary programs, all parameterized via a shell script and shipped
in a Docker image, in order to periodically backup a shared block store and remove
files according to their modify time.

The docker image is called `gcr.io/onec-co/datamon-datamover` and is tagged with
versions just as the Kubernetes sidecar, `v<release_number>`, where `v0.7` is the first
tag that will apply to the Datamover.

The `datamover` image contains two shell wrappers, `backup` and `datamover`.
Both fulfill approximately the same purpose, backing up files from an NFS share
to datamon.  The main difference is that `backup` uses standard *nix utils,
while `datamover` uses an auxilliary util maintained alongside datamon.
Their respective parameters are as follows:

## `backup`

* `-d` backup directory.  required if `-f` not present.
  this is the recommended way to specify files to backup from a kubernetes job.
* `-f` backup filelist.  list of files to backup.
* `-u` unlinkable filelist.  when specified, files that can be safely deleted
  after the backup are written to this list.  when unspecified, files are deleted
  by `backup`.
* `-t` set to `true` or `false` in order to run in test mode, which at present
  does nothing more than specify the datamon repo to use.


## `datamover`

* `-d` backup directory.  required.
* `-l` bundle label.  defaults to `datamover-<timestamp>`
* `-t` timestamp filter before.  a timestamp string in system local time among several formats, including
  - `<Year>-<Month>-<Day>` as in `2006-Jan-02`
  - `<Year><Month><Day><Hour><Minute>` as in `0601021504`
  - `<Year><Month><Day><Hour><Minute><second>` as in `060102150405`
  defaults to `090725000000`
* `-f` filelist directory.  defaults to `/tmp` and is the location to write
  - `upload.list`, the files that datamon will attempt to upload as part of the backup
  - `uploaded.list`, the files that have been successfully uploaded as part of the backup
  - `removable.list`, the files that have been successfully uploaded and that have a modify time before the specified timestamp filter
* `-c` concurrency factor.  defaults to 200.  tune this down in case of the NFS being hammered by too many reads during backup.
* `-u` unlink, a boolean toggle.  whether to unlink the files in `removeable.list` as part of the `datamover` script.  defaults to off/false/not present.

