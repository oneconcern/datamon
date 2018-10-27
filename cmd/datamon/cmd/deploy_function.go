package cmd

import (
	"fmt"
	"github.com/oneconcern/datamon/pkg/config"
	"github.com/oneconcern/datamon/pkg/kubeless"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
)

// branchCmd represents the branches command
var functionCmd = &cobra.Command{
	Use:   "function",
	Short: "Commands to deploy functions on Kubernetes.",
	Long: `Commands to deploy functions on Kubernetes.
`,

	Run: func(cmd *cobra.Command, args []string) {
		configFile, err := cmd.Flags().GetString("from-file")
		if configFile == "" {
			 log.Fatal("from-file attribute is empty ")
		}
		log.Printf("File %v submitted by user ", configFile)
		if err != nil {
			log.Fatalln(err)
		}

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
	deployCmd.AddCommand(functionCmd)
	functionCmd.Flags().StringP("from-file", "f", "", "Specify config file ")

}

