# Using datamon on cluster resources

### Disclaimer
At this moment, this configuration is tested locally on minikube on a linux box.

Making sure RBAC is enabled: `kubectl cluster-info dump | grep authorization-mode`

Further work is needed to _actually_ restrict access to this setup and make it truly private.

Further work is needed to define resources, roles etc. for our dev cluster.

Most importantly, moving further on requires setting some limitations on the default roles:

`kubernetes.io/bootstrapping=rbac-defaults`

### Use-case
A user wants to deploy kubernetes pods on some development
cluster to run `datamon`.

We assume this is an interactive session, e.g. using `kubectl exec -i -t ...` commands
to drive the `datamon` CLI running on the pod.

Other resources, such as jobs, etc. may be created to extend the basic use case.

### Scoping

The user should have its own "playground" namespace created.

Ex:
```
kubectl create namespace frederic-oneconcern-com
```

> **NOTE**:  this operation should normally be carried out by an admin.
> namespaces names cannot contain `@` or `.`

There are some role and binding to create and associate to this namespace.
```
kubectl create -f namespace/rbac.yaml
```
The user should define a context to run privately in this namespace.

Further we need to define a named user based on gcloud credentials (default gcloud access identifies to the cluster with a service account).

Ex: 
```
kubectl apply -f private-config.yaml
```
See our [sample config for minikube](./chart/namespace/private-config.yaml)

or from the CLI:
```
kubectl config set users.frederic@oneconcern.com.(...) # each user key must be entered separately...
kubectl config set-context private-frederic --cluster minikube --user frederic@oneconcern.com --namespace frederic-oneconcern-com
```

Set the new context as current:
```
kubectx private-frederic
```

> **NOTE**: `kubectx` is a small context-switching utility you may get from [here](https://github.com/ahmetb/kubectx)

### Container

During this experimental stage, we do not store the image on gcr.io.

Build the image locally into your minikube docker:
```
cd $(git rev-parse --show-toplevel)
eval $(minikube docker-env)
docker build -t datamon-pod:2.0.0 -f pod.Dockerfile .
```

### Authenticate

You must first sign-in to gcloud:

gcloud auth login
gcloud auth application-default login --scopes https://www.googleapis.com/auth/cloud-platform,email,profile

> **NOTE**: asking for proper scopes allows datamon to sign data with your actual name, not just your email

Create a secret in the dedicated namespace:

```
CONFIG_DIR=${HOME}/.config/gcloud
kubectl create secret generic my-creds --from-file=${CONFIG_DIR}/application_default_credentials.json
```

> **NOTE**: config is stored at some other place on MacOS

### Deploy using helm

```
cd $(git rev-parse --show-toplevel)/k8s/chart

helm install -n datamon-pod -f values.yaml .
```

> **NOTE**:
> This will require the tiller service account to get privileges enabled on the "private" namespace,
> including access to reading secrets.

This deploys a single pod running a minimal alpine linux equipped with datamon.

> **NOTE**:
> * it is possible to mount a specific datamon config file (specified in `values.yaml`).
> * it is possible to run multiple pods: this is actually a k8s deployment for which any number
>   of replicas may be specified.

At the moment, the pod is standalone and not very useful... It is possible to add extra volumes and mounts
to interact with storage resources locally.

### Working with the pod

Check the deployment did work well.
```
kubectl get pods

NAME                           READY   STATUS    RESTARTS   AGE
datamon-pod-7f696b48cd-rz77t   1/1     Running   0          36m
```

Connect to the pod via ssh.
``
kubectl exec -i -t datamon-pod-7f696b48cd-rz77t bash
```

Datamon config & credentials are mapped:
```
# ls .datamon/datamon.yaml
.datamon/datamon.yaml

# echo $GOOGLE_APPLICATION_CREDENTIALS
/home/project/.config/gcloud/application_default_credentials.json
```
> **NOTE**: several shells are available on this container (sh, bash, zsh, tcsh)

You can exercise datamon:
```
datamon repo create --repo fred-repo --description "yet another test repo"

datamon repo get --repo fred-repo
fred-repo , yet another test repo , Frédéric Bidon , frederic@oneconcern.com , 2019-10-30 10:01:08.407304538 +0000 UTC
```

### Unfolding...
```
helm delete --purge datamon-pod
```
