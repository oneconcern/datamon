# Datamon sidecar design

This document describes the current design of datamon sidecars.

<!-- Ideas to improve this design may be found [here](proposals/sidecar-design.md) -->

## Design

The current design favors the sidecar container approach, with bespoke signaling between containers,
over the CSI driver approach. Therefore,  datamon is not available as a kubernetes persistent volume plugin.

This design choice stems from the current inability by Kubernetes CSI API to handle ephemeral volumes.
Managing many short lived kubernetes volumes is at the moment not practical.

Signaling is implemented with files on a shared volume.

## Sidecar signaling

We need some coordination between the different containers running in the pod.

In particular, we need to keep the pod running and the sidecars to finish uploading
results when the main ARGO workflow is done processing.

Ensuring that data is ready for access (sidecar to main-container messaging)
as well as notification that the data-science program has
produced output data to upload (main-container to sidecar messaging),
is the responsibility of a few shell scripts shipped as part and parcel of the
Docker images that practicably constitute sidecars.

The coordination signaling defines the following protocol:

### File system mount (`fuse`)

| main(`wrap_application.sh`) | sidecar (`wrap_datamon.sh`) | what happens |
|-----------------------------|-----------------------------|--------------|
|                | <= mountdone  | application waits for input bundles to be mounted |
|                |               | (do some work...)                                 |
| initupload  => |               | datamon starts running the upload commands        |
|                | <= uploaddone | application waits until its output is archived    |

### Postgres SQL

A similar process is iterated through all configured datamon-postgres sidecars.

| main(`wrap_application.sh`) | sidecar (`wrap_datamon_pg.sh`) | what happens |
|-----------------------------|-----------------------------|--------------|
|                 | <= dbstarted    | application waits for the DB instance to be ready |
|                 |                 | (do some work...)                                 |
| initdbupload => |                 | datamon archives the database as a bundle         |
|                 | <= dbuploaddone | application waits until its output is archived    |

<!-- Internal notes -->
<!--
 While there's precisely one application container per Argo node,
a Kubernetes container created from an arbitrary image,
sidecars are additional containers in the same Kubernetes pod
-- or Argo DAG node, we can say, approximately synonymously --
that concert datamon-based data-ferrying setups with the application container.

> _Aside_: as additional kinds of data sources and sinks are added,
> we may also refer to "sidecars" as "batteries," and so on as semantic drift
> of the shell scripts shears away feature creep in the application binary.
-->

<!-- Internal notes -->
<!--
> _Aside_: historically, and in case it's necessary to roll back to an now-ancient
> version of the sidecar image, releases were tagged in git without the `v` prefix,
> and Docker tags prepended `v` to the git tag.
> For instance, `0.4` is listed on the github releases page, while
> the tag `v0.4` as in `gcr.io/onec-co/datamon-fuse-sidecar:v0.4` was used when writing
> Dockerfiles or Kubernetes-like YAML to accesses the sidecar container image.
-->

## Sidecar usage

Users need only place the `wrap_application.sh` script located in the root directory of the main container.
This
[can be accomplished](https://github.com/oneconcern/datamon/blob/master/hack/k8s/example-coord.template.yaml#L15-L24)
via an `initContainer` without duplicating version of the Datamon sidecar
image in both the main application Dockerfile as well as the YAML.
When using a block-storage GCS product, we might've specified a data-science application's
Argo DAG node with something like

```yaml
command: ["app"]
args: ["param1", "param2"]
```

whereas with `wrap_application.sh` in place, this would be something to the effect of

```yaml
command: ["/path/to/wrap_application.sh"]
args: ["-c", "/path/to/coordination_directory", "-b", "fuse", "--", "app", "param1", "param2"]
```

That is, `wrap_application.sh` has the following usage

```shell
wrap_application.sh -c <coordination_directory> -b <sidecar_kind> -- <application_command>
```

where
* `<coordination_directory>` is an empty directory in a shared volume
  (an
  [`emptyDir`](https://kubernetes.io/docs/concepts/storage/volumes/#emptydir)
  using memory-backed storage suffices).  each coordination directory (not necessarily the volume)
  corresponds to a particular DAG node (i.e. Kubernetes pod) and vice-versa.
* `<sidecar_kind>` is in correspondence with the containers specified in the YAML
  and may be among
  - `fuse`
  - `postgres`
* `<application_command>` is the data-science application command exactly as it
  would appear without the wrapper script.  That is, the wrapper script, relies the
  [conventional UNIX syntax](http://zsh.sourceforge.net/Guide/zshguide02.html#l11)
  for stating that options to a command are done being declared.

Meanwhile, each sidecar's datamon-specific batteries have their corresponding usages.

##### `gcr.io/onec-co/datamon-fuse-sidecar` -- `wrap_datamon.sh`

Provides filesystem representations (i.e. a folder) of [datamon bundles](#data-modeling).
Since bundles' filelists are serialized filesystem representations,
the `wrap_datamon.sh` interface is tightly coupled to that of the self-documenting
`datamon` binary itself.

```shell
./wrap_datamon.sh -c <coord_dir> -d <bin_cmd_I> -d <bin_cmd_J> ...
```

* `-c` the same coordination directory passed to `wrap_application.sh`
* `-d` all parameters, exactly as passed to the datamon binary, except as a
  single scalar (quoted) parameter, for one of the following commands
  - `config` sets user information associated with any bundles created by the node
  - `bundle mount` provides sources for data-science applications
  - `bundle upload` provides sinks for data-science applications

Multiple (or none) `bundle mount` and `bundle upload` commands may be specified,
and at most one `config` command is allowed so that an example `wrap_datamon.sh`
YAML might be

```yaml
command: ["./wrap_datamon.sh"]
args: ["-c", "/tmp/coord", "-d", "config create", "-d", "bundle upload --path /tmp/upload --message \"result of container coordination demo\" --repo ransom-datamon-test-repo --label coordemo", "-d", "bundle mount --repo ransom-datamon-test-repo --label testlabel --mount /tmp/mount --stream"]
```

or from the shell

```shell
./wrap_datamon.sh -c /tmp/coord -d 'config create' -d 'bundle upload --path /tmp/upload --message "result of container coordination demo" --repo ransom-datamon-test-repo --label coordemo' -d 'bundle mount --repo ransom-datamon-test-repo --label testlabel --mount /tmp/mount --stream'
```

----

_Aside on serialization format_

Each of these environment variables each contain a serialized dictionary
according the the following format

```
<entry_sperator><key_value_seperator><entry_1><entry_seperator><entry_2>...
```

where `<entry_sperator>` and `<key_value_seperator>` are each a single
character, anything other than a `.`, and each `<entry>` is of one of
two forms, either `<option>` or `<option><key_value_seperator><arg>`.

So for example

```
;:a;b:c
```

expresses something like a Python map

```python
{'a': True, 'b' : 'c'}
```

or shell option args

```
<argv0> -a -b c
```
