**Version: dev**

## datamon config create

Create a config

### Synopsis

Create a config to use for datamon to hold flags that do not
change.

The configuration file will be placed in $HOME/.datamon2/datamon.yaml

```
datamon config create [flags]
```

### Examples

```
# Replace path to gcloud credential file. Use absolute path
% datamon config create --credential /Users/ritesh/.config/gcloud/application_default_credentials.json,

# Replace path to gcloud credential file (use absolute path here)
% datamon config create --credential /Users/ritesh/.config/gcloud/application_default_credentials.json

# Specify a config bucket to store context details
% datamon config create --config fred-datamon-config --context test-context

```

### Options

```
      --config string       Set the config backend store to use (do not set the scheme, e.g. 'gs://')
      --context string      Set the context for datamon (default "dev")
      --credential string   The path to the credential file
  -h, --help                help for create
```

### Options inherited from parent commands

```
      --upgrade   Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon config](datamon_config.md)	 - Commands to manage a config

