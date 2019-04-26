package cmd

import (
	"io/ioutil"
	"os"
	"os/user"

	"gopkg.in/yaml.v2"

	"github.com/spf13/cobra"
)

var configGen = &cobra.Command{
	Use:   "create",
	Short: "Create a config",
	Long:  "Create a config to use for datamon. Config file will be placed in $HOME/.datamon/datamon.yaml",
	Run: func(cmd *cobra.Command, args []string) {
		user, err := user.Current()
		if user == nil || err != nil {
			logFatalln("Could not get home directory for user")
		}
		config := Config{
			Email:      repoParams.ContributorEmail,
			Name:       repoParams.ContributorName,
			Metadata:   repoParams.MetadataBucket,
			Blob:       repoParams.BlobBucket,
			Credential: credFile,
		}
		o, e := yaml.Marshal(config)
		if e != nil {
			logFatalln(e)
		}
		_ = os.Mkdir(user.HomeDir+"/.datamon", 0700)
		err = ioutil.WriteFile(user.HomeDir+"/.datamon/datamon.yaml", o, 0600)
		if err != nil {
			logFatalln(err)
		}
	},
}

func init() {

	requiredFlags := []string{addContributorEmail(configGen)}
	requiredFlags = append(requiredFlags, addContributorName(configGen))
	addBucketNameFlag(configGen)
	addBlobBucket(configGen)
	addCredentialFile(configGen)

	for _, flag := range requiredFlags {
		err := configGen.MarkFlagRequired(flag)
		if err != nil {
			logFatalln(err)
		}
	}

	configCmd.AddCommand(configGen)
}
