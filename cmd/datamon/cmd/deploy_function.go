package cmd

import (
	"github.com/oneconcern/datamon/pkg/config"
	"github.com/oneconcern/datamon/pkg/kubeless"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
)

// functionCmd represents the function create command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Commands to deploy functions on Kubernetes.",
	Long: `Commands to deploy functions on Kubernetes.
`,

	Run: func(cmd *cobra.Command, args []string) {
		configFile, err := cmd.Flags().GetString("config")
		if err != nil {
			log.Fatalln(err)
		}
		if configFile == "" {
			 log.Fatal("from-file attribute is empty ")
		}
		log.Printf("deploying using the %v config file ", configFile)


		configFileBytes, err := ioutil.ReadFile(configFile)
		if err != nil {
			log.Printf("Error while reading config file: %s", err)
		}

		var processor config.Processor
		err = yaml.Unmarshal(configFileBytes, &processor)
		if err != nil {
			log.Printf("Error parsing config file: %s ", err)
		}

		if len(processor.Content) == 0 {
			log.Fatalf("content attribute is empty ")
		}
		err = kubeless.ZipFile(processor.Content, processor.Name + ".zip")


	},
}

func init() {
	functionCmd.AddCommand(deployCmd)

}

