# Authentication

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
