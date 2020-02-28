package cmd

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var configGen = &cobra.Command{
	Aliases: []string{"create"},
	Use:     "set",
	Short:   "Create a local config file",
	Long: `Creates a local config file and sets the config value to use for datamon to hold flags that do not change, like remote config bucket or current context to use.

	By default, this configuration file will be placed in ` + configFileLocation(false) + `.

	Use the DATAMON_CONFIG environment variable to change this default target.
	`,
	Example: `# Replace path to gcloud credential file. Use absolute path
% datamon config set --credential /Users/ritesh/.config/gcloud/application_default_credentials.json,
config file created in /Users/ritesh/.datamon2/datamon.yaml

# Replace path to gcloud credentials file (use absolute path here)
% datamon config set --credential /Users/ritesh/.config/gcloud/application_default_credentials.json
config file created in /Users/ritesh/.datamon2/datamon.yaml

# Specify a config bucket and context
% datamon config set --config fred-datamon-config --context test-context
config file created in /Users/ritesh/.datamon2/datamon.yaml

# Switch context
% datamon config set --context another-context
config file created in /Users/ritesh/.datamon2/datamon.yaml

# Generate config in some non-default location
% ` + envConfigLocation + `=~/.config/.datamon/config.yaml datamon config set --config "remote-config-bucket"
config file created in /Users/ritesh/.config/.datamon/config.yaml
`,
	Run: func(cmd *cobra.Command, args []string) {
		optionInputs := newCliOptionInputs(config, &datamonFlags)
		_, err := optionInputs.contributor()
		if err != nil {
			wrapFatalln("failed to resolve contributor", err)
			return
		}

		localConfig := CLIConfig{
			Config:     datamonFlags.core.Config,
			Context:    datamonFlags.context.Descriptor.Name,
			Credential: datamonFlags.root.credFile,
			Metrics:    datamonFlags.root.metrics,
		}

		file := configFileLocation(true)

		if ext := filepath.Ext(file); ext != ".yaml" {
			infoLogger.Printf("warning: the generated config file will contain a yaml document, but the file extension is %q", ext)
		}
		o, err := localConfig.MarshalConfig()
		if err != nil {
			wrapFatalln("could not serialize config to yaml", err)
			return
		}

		err = os.Mkdir(filepath.Dir(file), 0777)
		if err != nil && !os.IsExist(err) {
			wrapFatalln("could not create directory to hold config "+filepath.Dir(file), err)
			return
		}

		err = ioutil.WriteFile(file, o, 0600)
		if err != nil {
			wrapFatalln("error writing config file "+file, err)
			return
		}

		log.Printf("config file created in %s", file)
	},
}

func init() {
	addCredentialFile(configGen)
	configCmd.AddCommand(configGen)
}
