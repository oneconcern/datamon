**Version: dev**

## datamon config set

Create a local config file

### Synopsis

Creates a local config file and sets the config value to use for datamon to hold flags that do not change, like remote config bucket or current context to use.

	By default, this configuration file will be placed in $HOME/.datamon2/datamon.yaml.

	Use the DATAMON_CONFIG environment variable to change this default target.
	

```
datamon config set [flags]
```

### Examples

```
# Replace path to gcloud credential file. Use absolute path
% datamon config set --credential /Users/ritesh/.config/gcloud/application_default_credentials.json,
config file created in /Users/ritesh/.datamon2/datamon.yaml

# Replace path to gcloud credentials file (use absolute path here)
% datamon config set --credential /Users/ritesh/.config/gcloud/application_default_credentials.json
config file created in /Users/ritesh/.datamon2/datamon.yaml

# Specify a config bucket to store context details
% datamon config set --config fred-datamon-config --context test-context
config file created in /Users/ritesh/.datamon2/datamon.yaml

# Generate config in some non-default location
% DATAMON_CONFIG=~/.config/.datamon/config.yaml datamon config set --config "remote-config-bucket"
config file created in /Users/ritesh/.config/.datamon/config.yaml

```

### Options

```
      --config string       Set the config backend store to use (bucket name: do not set the scheme, e.g. 'gs://')
      --context string      Set the context for datamon (default "dev")
      --credential string   The path to the credential file
  -h, --help                help for set
```

### Options inherited from parent commands

```
      --upgrade   Upgrades the current version then carries on with the specified command
```

### SEE ALSO

* [datamon config](datamon_config.md)	 - Commands to manage the config file

