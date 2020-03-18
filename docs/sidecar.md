# Datamon sidecar guide

## Goal

The objective is to expose persistent data at rest to a consuming application running in a kubernetes pod,
leveraging datamon's features:
* data versioned and tags as whole consistent data sets
* high throughput, with parallel I/Os
* deduplicated storage, to further improve I/O performance

Consumed inputs are downloaded, then mounted as read-only volumes.
After the application has produced its outputs, the sidecar container uploads the results to GCS as datamon bundles.

This data presentation layer is part of the "batteries" feature.

## Use-cases

The sidecar implementation is primarily intended to be used by ARGO workflows.

* The ARGO controller starts a pod to run a data science application in the "main container".
* Workflow settings specify some input and output volumes
* When the application is done, the output is persisted
* The controller stops the pod by signaling all containers

ARGO brings native support to carry out this process for `S3` and `GCS` storage buckets.
The datamon sidecar feature is about augmenting this to support datamon-backed storage (e.g. on top of `GCS`),
including database servers.

> **NOTE**: the sidecar utility does make any assumption about the pod's environment and may be used with any
> containerized environment.

### Working with data sets stored as plain files

This use case is addressed by the "datamon-fuse-sidecar" container.

* Both inputs and outputs are specified as _volume mounts_ with their corresponding datamon location:
  (context, repo, bundle or label)
* Datamon mounts input as _read-only_ fuse file systems, exposed to all containers in the pod
* Outputs are plain volumes (provisioned with sufficient local disk space) which are _uploaded_ by datamon upon
  completion of the main application


### Working with data sets stored in a Postgres RDBMS

This use case is addressed by the "datamon-pg-sidecar" container.

* Both inputs and outputs are specified as _Postgres database servers_ (i.e a port)
* Database files are first downloaded by datamon, then a Postgres server is spun up in the sidecar container
* A staging volume has to be provisioned with sufficient local disk space to hold the database
* When the main application completes, the sidecar takes over, shuts down the database then upload the files
  to the target datamon location (context, repo[, label])

## Sidecars features and limitations

More on the [design of datamon sidecars](sidecar-design.md).

### Application wrapper

The application wrapper is a small Bourne shell script that wraps around your main application to do 3 things:
1. Wait until all declared input data is available
2. Run the application
3. Wait until all declared output data is persisted

The wrapper terminates when the last step is completed.
This coordination is achieved thanks to "signaling files" written on a volume shared by all containers on the same pod.

#### How to set up a wrapper?

The wrapper script is available as a docker image `gcr.io/onec-co/datamon-wrapper`. The image is very small and contains just this script and
a minimal alpine Linux distribution to be able to copy it.

This script is typically retrieved using an `initContainer`, like this:
```yaml
  # The initContainer retrieves the wrap_application.sh script and makes it
  # available to other application containers.
  initContainers:
    - name: init-application-wrap
      image: "gcr.io/onec-co/datamon-wrapper:latest"
      imagePullPolicy: Always
      command: ["sh", "-c", "cp -a /.scripts/* /scripts"]
      volumeMounts:
        - mountPath: /scripts
          name: application-wrapper
```

The main application container would then use the script like this:
```yaml
  containers:
    - name: demo-app
      image: "gcr.io/onec-co/my-app:latest"
      imagePullPolicy: Always
      command: ["/scripts/wrap_application.sh"]
      args:
        - "-c"  # specifies the location for coordination messages
        - "/tmp/coord"
        - "-b"  # specifies the coordination type used (fuse|postgres), each type following a specific coordination scheme
        - "postgres"
        - "-d"  # when postgres is used, specifies the databases to be waited for (space separated list of configured "names" for db server instances)
        - "db1 db2 db3"
        - "--"
        - "mock_application_pg.sh"  # the application to be wrapped and its parameters (none for this mock)
      volumeMounts:
        - mountPath: /scripts
          name: application-wrapper
        - mountPath: /tmp/coord
          name: container-coord
```
#### How does it work

1. Coordination signals are received from the sidecars and waited for by the wrapper
2. The wrapper waits for the application to terminate then signals the sidecars to start uploading
3. Waits for sidecars to signal all uploads are complete

#### Parameters

| Argument flag | Default value   | Description                                                |
|---------------|-----------------|------------------------------------------------------------|
| `-c`          | N/A (required)  | Mount point for signaling files                            |
| `-b`          | N/A (required)  | Coordination type (`fuse|postgres`)                        |
| `-d`          | N/A             | Database server names - required when `-b postgres` is set |

All flags after `--` are passed to the main application.

### Fuse sidecar

* a pod may start only one fuse sidecar, this sidecar may mount several read-only volumes
*
* for every read-only mounted volume
#### Limitations
* a single volume specification cannot specify both datamon source and destination

### Postgres sidecar

* a pod may start several Postgres sidecars, every sidecar will spin up a single Postgres database server
* every database server _may_ expose several logical databases on the same port
* every sidecar exposes a postgres port, accessible to all containers on the pod. Ports must be unique.
* when several database servers are declared, all will be downloaded before the application actually starts
* the application is free to access the database server as super-user (user `postgres`) or any predefined database user
* a postgres sidecar without a source repo will create a new database from scratch
* a postgres sidecar without a destination repo will skip the upload stage: any changes to data will be lost. This is useful to use read-only databases.

#### Limitations

* at the moment, the fuse sidecar only works with `streamed` fuse read-only mounts. This means that the mount is available very quickly and that files
  are downloaded on demand, without any staging area to provision. Memory cache options in this mode are not configurable (default cache size: 50 MB per mounted volume).
* we don't support fuse AND postgres coordination at the same time with a single wrapper (possible with nested wrappers)
* we don't support running several databases in one single sidecar container
* the staging area to download the database must be provisioned with sufficient disk to hold the data files
* errors caught during the download or upload phases are not handled and result in the application waiting indefinitely
* postgres databases are backed as plain files. This is way faster than carrying out a logical export but exposes us
  to compatibility issues whenever a new major postgres version is issued. At this moment, sidecars work with Postgres 12.2,
  meaning that a migration operation will have to be carried out when we want to upgrade the sidecar containers to Postgres 13.

## Parameters and default values

### Datamon parameters

Sidecars run datamon the way you would use it locally: defining a local configuration file is strongly advised.
The process runs as non-root use `developer`. The default location of the datamon config file is `/home/developer/.datamon2/datamon.yaml`

Default settings point to the "workshop context" buckets. The sample pod specification below exhibits how to map a configuration for your datamon context.

Another way of doing this is to use environment variables `DATAMON_GLOBAL_CONFIG` (location of the main metadata bucket) and `DATAMON_CONTEXT` (which context is used).

Sample config resource for a pod:
```yaml
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: datamon-local-config
  labels:
    app: datamon-coord-pg-demo
data:
  # the buckets configuration (context) for datamon
  datamon.yaml: |
    config: workshop-config
    context: dev
```

### fuse sidecar

#### Parameters

| Environment                     |   YAML key              | Default value          | Description                                                        |
|---------------------------------|-------------------------|------------------------|--------------------------------------------------------------------|
| `dm_fuse_params`                | N/A                     | N/A (require)          | Location of YAML config file for sidecar                           |

> **NOTE**: it is possible to configure the sidecar via environment variables: this is reserved or internal and debug usage

#### Sample configuration for fuse sidecar

**Example:** 1 input (read-only), 1 output (to be saved)

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: datamon-fuse-params
  labels:
    app: datamon-coord-fuse-demo
data:
  fuse-params.yaml: |
    globalOpts:
      coordPoint: /tmp/coord
      # The ones below are no more used (see datamon.yaml above) but kept because of sidecar_param requirements
      configBucketName: datamon-config-test-sdjfhga
      contextName: datamon-sidecar-test
    bundles:
      - name: src
        # mount point for the volume containing read-only data
        srcPath: /tmp/mount
        # datamon location specification: (repo, bundle|label)
        srcRepo: ransom-datamon-test-repo
        # identify the desired point in time with either label or bundleID (not both)
        srcLabel: testlabel
        srcBundle: ""
      - name: dest
        # mount point for the volume containing data to be saved
        destPath: /tmp/upload
        # datamon location specification on how to save data: (repo, bundle|label)
        destRepo: ransom-datamon-test-repo
        # the commit message for the saved bundle
        destMessage: result of container coordination demo
        destLabel: coordemo
```

### Postgres sidecar

#### Parameters
| Environment                     |   YAML key              | Default value          | Description                                                        |
|---------------------------------|-------------------------|------------------------|--------------------------------------------------------------------|
| `SIDECAR_CONFIG`                | N/A                     | `/config/pgparams.yaml`| Location of YAML config file for sidecar                           |
| `SIDECAR_GLOBALOPTS_COORD_POINT`| `globalOpts.coordPoint` | `/tmp/coord`           | The mount point used for signaling                                 |
| `SIDECAR_DATABASE_NAME`         | `database.name`         | `db`                   | db server name used to dispatch signals                            |
| `SIDECAR_DATABASE_DATADIR`      | `database.dataDir`      | `/pg_stage`            | The mount point used to download the db                            |
| `SIDECAR_DATABASE_PGPORT`       | `database.pgPort`       | `5432`                 | The postgres port exposed                                          |
| `SIDECAR_DATABASE_SRCREPO`      | `database.srcRepo`      |                        | The repo the db is downloaded from (if none, a new db is created)  |
| `SIDECAR_DATABASE_SRCBUNDLE`    | `database.srcBundle`    |                        | The bundle defining the point in time the db is downloaded from    |
| `SIDECAR_DATABASE_SRCLABEL`     | `database.srcLabel`     |                        | The label defining the point in time the db is downloaded from     |
| `SIDECAR_DATABASE_DESTREPO`     | `database.destRepo`     |                        | The repo the db is uploaded to (if none, the db is not saved)      |
| `SIDECAR_DATABASE_DESTLABEL`    | `database.destLabel`    |                        | The label put on the uploaded db (optional)                        |
| `SIDECAR_DATABASE_DESTMESSAGE`  | `database.destMessage`  |                        | The commit message for the saved bundle (required)                 |
| `SIDECAR_DATABASE_OWNER`        | `database.owner`        |                        | When a database is created from scratch, user to create (optional) |

> **NOTES**:
> - for source specification, either bundle or label may be used, but not both
> - for sidecar configuration as yaml, do not use inline comments (e.g. `mykey: value #<-- yaml comment`)
> - the keys `globalOpts.sleepInsteadOfExit` (default: `"false"`) and `globalOpts.sleepTimeout` (default: `600` sec) are intended for internal use and debug only
>   not for production usage (makes the sidecar sleep for a while after completion)

#### Sample configurations for Postgres sidecar

We strongly suggest that you use config map objects and declare parameters using this YAML config. Here are some examples. You may see these examples in action with our demo app.
The full code of the working demo is available here: https://github.com/oneconcern/datamon/tree/master/hack/fuse-demo and here https://github.com/oneconcern/datamon/tree/master/hack/k8s

**Example 1:**  write only db, created from scratch, then saved after the application completes
```yaml
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: datamon-pg-sidecar-config-1
  labels:
    app: datamon-coord-pg-demo
data:
  pgparams.yaml: |
    globalOpts:
      coordPoint: /tmp/coord
    database:
      name: db1
      pgPort: "5430"
      # sidecar will create dbuser with super-user privileges (alternative to user "postgres")
      owner: dbuser
      # no src: means we want to create this from scratch
      # dest: means we want to save after our work is done
      destRepo: example-repo
      destMessage: postgres sidecar coordination example (write only)
      destLabel: write only example
```

**Example 2:** retrieve db from an existing bundle then save changes as another bundle
```yaml
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: datamon-pg-sidecar-config
  labels:
    app: datamon-coord-pg-demo
data:
  pgparams.yaml: |
    globalOpts:
      coordPoint: /tmp/coord
    database:
      name: db2
      pgPort: "5429"
      srcRepo: example-repo
      srcLabel: my-desired-point-in-time
      destRepo: example-repo
      destMessage: postgres sidecar coordination example (read-write)
      destLabel: read-write example
```

**Example 3:** retrieve db from an existing bundle to use for reading only (no changes saved)

Notice that the database is not really _read-only_: the application may indeed modify the db content. Changes will just not be saved after the app exits.

```yaml
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: datamon-pg-sidecar-config
  labels:
    app: datamon-coord-pg-demo
data:
  # - a read only instance created from existing bundle, then ditched
  pgparams.yaml: |
    globalOpts:
      coordPoint: /tmp/coord
    database:
      name: db3
      pgPort: "5428"
      srcRepo: example-repo
      srcLabel: another-desired-point-in-time
      # no dest: means the upload signal just shuts down the db, and no upload is actually carried out
```

## Walking through a full example

The full example is based on the demo. For fuse: [k8s spec](https://github.com/oneconcern/datamon/blob/master/hack/k8s/example-coord.template.yaml).
For Postgres: [k8s spec](https://github.com/oneconcern/datamon/blob/master/hack/k8s/example-coord-pg.template.yaml).

Notice that the demo templates introduce some variability to properly run on CI.

Our demo uses a kubernetes `Deployment` to deploy a pod. This not a requirement, though, and you may use ARGO resources or kubernetes standard resources such
as `jobs` or `statefulSets`.

### Init container

We need to retrieve the wrapper script into some shared volume.
This step remains the same, whether you plan to use the fuse or postgres flavor of the sidecar.

```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment
  namespace: datamon-ci
spec:
  selector:
    matchLabels:
      app: datamon-coord-fuse-demo
      instance: v0.0.0
  replicas: 1
  template:
    metadata:
      labels:
        app: datamon-coord-fuse-demo
        instance: v0.0.0
    spec:
      initContainers:
      - name: init-application-wrap
        image: gcr.io/onec-co/datamon-wrapper:latest
        imagePullPolicy: Always
        command: ["sh", "-c", "cp -a /.scripts/* /scripts"]
        volumeMounts:
        - mountPath: /scripts
          name: application-wrapper

...

      volumes:
      - name: application-wrapper
        emptyDir: {}
```

### Config maps

Using config maps is the recommended way to setup datamon sidecars.
We need:

1. a config map to configure datamon (config bucket, context)
2. plain files: a single config map to configure a datamon-fuse sidecar
3. postgres: a config map for each datamon-pg sidecar

See the examples provided above for these configs.

Config maps are declared as volumes on the pod, like this.

Fuse example (one single config):
```yaml
      volumes:
      - name: application-wrapper
        emptyDir: {}
      - name: fuse-params
        configMap:
          name: datamon-fuse-params
          defaultMode: 0555
      - name: datamon-config
        configMap:
          name: datamon-local-config
          defaultMode: 0555
```

Postgres example (deploys 3 different configs for this demo):
```yaml
      volumes:
      - name: application-wrapper
        emptyDir: {}
      - name: pg-config-1
        configMap:
          name: datamon-pg-sidecar-config-1
          defaultMode: 0555
      - name: pg-config-2
        configMap:
          name: datamon-pg-sidecar-config-2
          defaultMode: 0555
      - name: pg-config-3
        configMap:
          name:datamon-pg-sidecar-config-3
          defaultMode: 0555
      - name: datamon-config
        configMap:
          name: datamon-local-config
          defaultMode: 0555
```

### Planning for storage requirements

> **NOTE**: this part is not included with the demo files, which use pod local storage only.

#### Plain files: fuse sidecar
We need to provision the storage needed before upload

#### Database: Postgres sidecar
We need to provision the storage needed before download (upload is carried out from the same volume)


We do that with a `PersistentVolumeClaim` and a `PersistentVolume`.
The volume will be relinquished by kubernetes when the pod exits.

Declare as many claims and volumes as you need.

#### Persistent volume claims

```yaml
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: my-app-staging-claim
spec:
  storageClassName: ssd
  persistentVolumeReclaimPolicy: Delete
  finalizers: []
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
```

```yaml
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: my-app-staging-volume
  type: pd-ssd
spec:
  storageClassName: ssd
  persistentVolumeReclaimPolicy: Delete
  capacity:
    storage: 10Gi
  accessModes:
    - ReadWriteOnce
```

### Wrapping the main application

#### Plain files: fuse sidecar
Let's assume your app is containerized. The upload staging volume is attached to the main application.

Running the container with the wrapper looks something like this.

```yaml
      containers:
      - name: demo-app
        image: "gcr.io/onec-co/my-app:latest"
        imagePullPolicy: Always
        command: ["/scripts/wrap_application.sh"]
        args:
        - "-c"  # specifies the location for coordination messages
        - "/tmp/coord"
        - "-b"  # specifies the coordination type used (fuse|postgres), each type following a specific coordination scheme
        - "fuse"
        - "--"
        - "mock_application.sh"
        - "/tmp/mount"   # volume where the files are mounted (read-only)
        - "/tmp/upload"  # staging area to upload
        volumeMounts:
        - mountPath: /scripts
          name: application-wrapper
        - mountPath: /tmp/coord
          name: container-coord
        - mountPath: /tmp/upload
          name: upload-source
        - mountPath: /tmp/mount
          name: fuse-mountpoint

...

      volumes:
      - name: application-wrapper
        emptyDir: {}
      - name: container-coord
        emptyDir: {}
      - name: upload-source
        persistentVolumeClaim:
          claimName: my-app-staging-claim
```

#### Database: Postgres sidecar

The app container does not need to mount the staging volume: only a pg sidecar needs it.
```yaml
      volumes:
      - name: application-wrapper
        emptyDir: {}
      - name: container-coord
        emptyDir: {}
      - name: staging-area-1         # database volume: provision sufficient disk for this operation
        persistentVolumeClaim:
          claimName: my-app-staging-claim-1
      - name: staging-area-2         # database volume: provision sufficient disk for this operation
        persistentVolumeClaim:
          claimName: my-app-staging-claim-2
      - name: staging-area-3         # database volume: provision sufficient disk for this operation
        persistentVolumeClaim:
          claimName: my-app-staging-claim-3
```

### Configuring sidecars

#### Plain files: fuse sidecar
The sidecar needs access to both the coordination mount point (read and written by the wrapper) and the upload volume (written by the app).

```yaml
      containers:
....
      - name: datamon-sidecar
        image: "gcr.io/onec-co/datamon-fuse-sidecar:latest"
        imagePullPolicy: Always
        command: ["wrap_datamon.sh"]
        args: []
        securityContext:
          privileged: true
        volumeMounts:
        - mountPath: /tmp/upload
          name: upload-source
        - mountPath: /tmp/coord
          name: container-coord
        - mountPath: /tmp/mount
          name: fuse-mountpoint
          mountPropagation: "Bidirectional"
        - mountPath: /tmp/gac
          name: google-application-credentials
        - mountPath: /config
          name: fuse-params
        - mountPath: /home/developer/.datamon2
          name: datamon-config
        env:
        - name: dm_fuse_params
          value: /config/fuse-params.yaml


      volumes:
      - name: fuse-mountpoint
        emptyDir: {}
      - name: application-wrapper
        emptyDir: {}
      - name: container-coord
        emptyDir: {}
      - name: upload-source
        persistentVolumeClaim:
          claimName: my-app-staging-claim
      - name: fuse-params
        configMap:
          name: datamon-fuse-params
          defaultMode: 0555
      - name: datamon-config
        configMap:
          name: datamon-local-config
          defaultMode: 0555
```

#### Database: Postgres sidecar
The sidecar needs access to the staging volume.

In this example, with spin up 3 sidecars, each illustrating a different use-case.

```yaml
      containers:
...
      # A datamon-pg-sidecar container spins up a postgres database retrieved from a datamon bundle
      - name: datamon-sidecar-1
        image: "gcr.io/onec-co/datamon-pg-sidecar:latest"
        imagePullPolicy: Always
        command: ["wrap_datamon_pg.sh"]
        args: []
        securityContext:
          privileged: true
        volumeMounts:
        - mountPath: /pg_stage
          name: staging-area-1        # volume where the database is mounted
        - mountPath: /tmp/coord
          name: container-coord       # shared volume for coordination beetwen application and sidecar
        - mountPath: /config          # sidecar parameters
          name: pg-config-1
        - mountPath: /home/developer/.datamon2
          name: datamon-config

      # Another database is spun, with a different use case
      - name: datamon-sidecar-2
        image: "gcr.io/onec-co/datamon-pg-sidecar:latest"
        imagePullPolicy: Always
        command: ["wrap_datamon_pg.sh"]
        args: []
        securityContext:
          privileged: true            # the container runs as non-root but needs to perform some sudo operations
        volumeMounts:
        - mountPath: /pg_stage
          name: staging-area-2        # volume where the database is mounted
        - mountPath: /tmp/coord
          name: container-coord       # shared volume for coordination beetwen application and sidecar
        - mountPath: /config          # sidecar parameters
          name: pg-config-2
        - mountPath: /home/developer/.datamon2
          name: datamon-config

      # Another database is spun, with again a different use case
      - name: datamon-sidecar-3
        image: "gcr.io/onec-co/datamon-pg-sidecar:$SIDECAR_TAG"
        imagePullPolicy: Always
        command: ["wrap_datamon_pg.sh"]
        args: []
        securityContext:
          privileged: true
        volumeMounts:
        - mountPath: /pg_stage
          name: staging-area-3        # volume where the database is mounted
        - mountPath: /tmp/coord
          name: container-coord       # shared volume for coordination beetwen application and sidecar
        - mountPath: /config          # sidecar parameters
          name: pg-config-3
        - mountPath: /home/developer/.datamon2
          name: datamon-config

...

      volumes:
      - name: application-wrapper
        emptyDir: {}
      - name: container-coord
        emptyDir: {}
      - name: staging-area-1         # database volume: provision sufficient disk for this operation
        persistentVolumeClaim:
          claimName: my-app-staging-claim-1
      - name: staging-area-2         # database volume: provision sufficient disk for this operation
        persistentVolumeClaim:
          claimName: my-app-staging-claim-2
      - name: staging-area-3         # database volume: provision sufficient disk for this operation
        persistentVolumeClaim:
          claimName: my-app-staging-claim-3
```

## Sidecar releases

Sidecar containers follow the same release cycle as datamon.

There are currently two sidecar images:

* `gcr.io/onec-co/datamon-fuse-sidecar` provides hierarchical filesystem access
* `gcr.io/onec-co/datamon-pg-sidecar` provides PostgreSQL database access

At this moment, these images remain part of our private `gcr.io` repository. You may look at them from here: https://console.cloud.google.com/gcr/images

Sidecars are versioned along with
[github releases](https://github.com/oneconcern/datamon/releases/)
of the [desktop binary](install.md).

Docker image tags follow the github releases.

See the latest release [here](https://github.com/oneconcern/datamon/releases/latest).

You embed a datamon sidecar in your kubernetes pod specification like this:
```yaml
spec:
  ...
  containers:
    - name: datamon-sidecar
    - image: gcr.io/onec-co/datamon-fuse-sidecar:v2.1.0
  ...
```
