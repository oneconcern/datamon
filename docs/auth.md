# Authentication

For non kubernetes use, gcloud credentials are forwarded by default.
Inside a kubernetes pod, Datamon will use kubernetes service credentials.

Starting with v2.0, datamon keeps track of who contributed what. The identity of contributors
is obtained from an OIDC identity provider (Google ID).

Make sure your gcloud credentials have been setup, with proper scopes (default gke credentials lack the email scope).

```bash
gcloud auth application-default login --scopes https://www.googleapis.com/auth/cloud-platform,email,profile
```

See the complete login procedure [below](login-to-google).

```bash
% cat ~/.datamon2/datamon.yaml
credential: /Users/ritesh/.config/gcloud/application_default_credentials.json
context: dev
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

## Login to google

All steps:
```bash 
gcloud auth login

gcloud config set compute/zone **********
gcloud config set project *******

gcloud auth application-default login --scopes https://www.googleapis.com/auth/cloud-platform,email,profile
```

You should have your credentials stored in `~/.config/gcloud/application_default_credentials.json`.

This is used by default by all Google libraries. You may change the location of the credential files
an specify the new file with the `GOOGLE_APPLICATION_CREDENTIALS` environment variable.

If you want to define a specific location for credential specifically for datamon, you set this location
in the `credential` key in the config. Letting this config empty will just fall back to defaults.

## Troubleshooting auth

Authentication against google may be subject to changes in the google API. Please report any problem.

Checking you gcloud version:
```
gcloud --version
Google Cloud SDK 296.0.1
...
```

Datamon has been succesfully tested using gcloud v.212, which is the version that ships on ubuntu 18 (bionic).
In particular, our CI currently uses this version.

### Breaking change with Google API SDK v292

https://cloud.google.com/sdk/docs/release-notes#29200_2020-05-12

Check your version `gcloud`: if it is more recent than v292, you are affected by this.

From gcloud v292 onwards, the above step must be modified as:
```
gcloud auth application-default login --disable-quota-project --scopes https://www.googleapis.com/auth/cloud-platform,email,profile
```

### Revoking your current credentials

You might want to logout and reconnect again.

```
gcloud auth revoke
```

### Known errors

#### could not create oauth service: google: could not find default credentials. See https://developers.google.com/accounts/docs/application-default-credentials for more information

You don't have credentials to connect to gcloud. First authenticate to gcloud using `gcloud auth`

#### could not retrieve userinfo: googleapi: Error 403: User must be authenticated when user project is provided, forbidden

This error starts to appear with v292. It requires disabling the quota project as shown above.

### Testing auth unitarily with your configuration

Datamon may be complex. In order to test and isolate pure authentication errors, you may use the following:

```
git clone https://github.com/oneconcern/datamon
cd datamon/pkg/auth/google
go test -v
```

This attempts to contact the google `userinfo` endpoint with your current credentials. No configuration is used: just defaults
or `GOOGLE_APPLICATION_CREDENTIALS`.

If this works, datamon should work without more configuration.
