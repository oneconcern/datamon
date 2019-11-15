package cmd

import (
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"github.com/spf13/cobra"
)

const datamonDir = ".datamon2"

var configGen = &cobra.Command{
	Use:   "create",
	Short: "Create a config",
	Long:  "Create a config to use for datamon. Config file will be placed in $HOME/" + datamonDir + "/datamon.yaml",
	Run: func(cmd *cobra.Command, args []string) {
		_, err := paramsToContributor(datamonFlags)
		if err != nil {
			wrapFatalln("contributor datamonFlags present", err)
			return
		}
		user, err := user.Current()
		if user == nil || err != nil {
			wrapFatalln("Could not get home directory for user", nil)
			return
		}
		config := CLIConfig{
			Config:     datamonFlags.core.Config,
			Context:    datamonFlags.context.Descriptor.Name,
			Credential: datamonFlags.root.credFile,
		}
		o, e := yaml.Marshal(config)
		if e != nil {
			wrapFatalln("serialize config to yaml", e)
			return
		}
		_ = os.Mkdir(filepath.Join(user.HomeDir, datamonDir), 0777)
		err = ioutil.WriteFile(filepath.Join(user.HomeDir, datamonDir, "datamon.yaml"), o, 0666)
		if err != nil {
			wrapFatalln("write config file", err)
			return
		}
	},
}

func init() {
	addCredentialFile(configGen)
	addContextFlag(configGen)
	addConfigFlag(configGen)
	configCmd.AddCommand(configGen)
}
