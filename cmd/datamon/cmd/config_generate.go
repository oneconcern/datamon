package cmd

import (
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"github.com/spf13/cobra"
)

var configGen = &cobra.Command{
	Use:   "create",
	Short: "Create a config",
	Long:  "Create a config to use for datamon. Config file will be placed in $HOME/.datamon/datamon.yaml",
	Run: func(cmd *cobra.Command, args []string) {
		_, err := paramsToContributor(params)
		if err != nil {
			wrapFatalln("contributor params present", err)
			return
		}
		user, err := user.Current()
		if user == nil || err != nil {
			wrapFatalln("Could not get home directory for user", nil)
			return
		}
		config := Config{
			Metadata:   params.repo.MetadataBucket,
			Blob:       params.repo.BlobBucket,
			Credential: params.root.credFile,
		}
		o, e := yaml.Marshal(config)
		if e != nil {
			wrapFatalln("serialize config to yaml", e)
			return
		}
		_ = os.Mkdir(filepath.Join(user.HomeDir, ".datamon"), 0777)
		err = ioutil.WriteFile(filepath.Join(user.HomeDir, ".datamon", "datamon.yaml"), o, 0666)
		if err != nil {
			wrapFatalln("write config file", err)
			return
		}
	},
}

func init() {
	addContributorEmail(configGen)
	addContributorName(configGen)
	addBucketNameFlag(configGen)
	addBlobBucket(configGen)
	addCredentialFile(configGen)

	configCmd.AddCommand(configGen)
}
