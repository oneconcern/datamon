package cmd

import (
	"bytes"
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/oneconcern/datamon/pkg/config"
	"github.com/oneconcern/datamon/pkg/kubeless"
	"github.com/oneconcern/datamon/pkg/storage/sthree"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"io/ioutil"
	"log"
	"os"
	"strings"
)

var configFile string

// functionCmd represents the function create command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Commands to deploy models on Kubernetes.",
	Long: `Commands to deploy models on Kubernetes.
`,

	Run: func(cmd *cobra.Command, args []string) {

		if configFile == "" {
			log.Fatal("from-file attribute is empty ")
		}
		log.Printf("deploying model using config file %s", configFile)

		configFileBytes, err := ioutil.ReadFile(configFile)
		if err != nil {
			log.Printf("Error while reading config file: %s", err)
		}

		err = viper.ReadConfig(bytes.NewBuffer(configFileBytes))
		if err != nil {
			log.Fatalln(err)
		}

		processor := config.Processor{}

		err = viper.Unmarshal(&processor)
		if err != nil {
			log.Printf("Error parsing config file: %s ", err)
		}

		if len(processor.Content) == 0 {
			log.Fatalf("content attribute is empty ")
		}
		zipfile, err := kubeless.ZipFile(processor.Content, processor.Name)
		if err != nil {
			log.Fatalf("create zip is failing. error: %v", err)
		}

		file, err := os.Open(zipfile)
		if err != nil {
			log.Fatalf("error in opening zip file %s. error %v ", zipfile, err)
		}

		bs := sthree.New(sthree.Bucket(*aws.String("oneconcern-datamon-dev")),
			sthree.AWSConfig(aws.NewConfig().WithRegion("us-west-2").WithCredentialsChainVerboseErrors(true)))

		s3FileName := zipfile[strings.LastIndex(zipfile, "/")+1:]

		log.Printf("uploading file %s to AWS S3 ", s3FileName)
		err = bs.Put(context.Background(), s3FileName, file)

		if err != nil {
			log.Fatalf("s3 upload [%s]: %v", processor.Name, err)
		}
	},
}

func init() {
	deployCmd.Flags().StringVarP(&configFile, "config", "c", "", "the output format to use")

	modelCmd.AddCommand(deployCmd)
}
