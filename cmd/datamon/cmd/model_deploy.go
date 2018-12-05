package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/oneconcern/kubeless/cmd/kubeless/function"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/oneconcern/datamon/pkg/config"
	"github.com/oneconcern/datamon/pkg/kubeless"
	"github.com/oneconcern/datamon/pkg/storage/sthree"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configFile string

// functionCmd represents the function create command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Commands to deploy models on Kubernetes.",
	Long:  "Commands to deploy models on Kubernetes.",

	Run: func(cmd *cobra.Command, args []string) {

		if configFile == "" {
			log.Fatal("from-file attribute is empty ")
		}
		log.Printf("deploying model using config file %s", configFile)

		configFileBytes, err := ioutil.ReadFile(configFile)
		if err != nil {
			log.Printf("Error while reading config file: %s", err)
		}

		ext, err := kubeless.ConfigExt(configFile)
		if err != nil {
			log.Fatal(err)
		}

		viper.SetConfigType(ext)
		viper.ReadConfig(bytes.NewBuffer(configFileBytes))
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
		out, err := exec.Command("aws", "s3", "presign", "s3://oneconcern-datamon-dev/"+s3FileName).Output()
		if err != nil {
			log.Fatal(err)
		}

		bucketUrl, err := url.QueryUnescape(fmt.Sprintf("%s", bytes.TrimRight(out, "\n")))
		if err != nil {
			log.Fatal(err)
		}
		err = function.DeployModel(processor.Name, "default", processor.Command[0], bucketUrl, "",
			processor.Runtime, "", processor.Resources.Mem.Max, processor.Resources.CPU.Max, "180", "", 8080,
			false, make([]string, 0), make([]string, 0), make([]string, 0))

		if err != nil {
			log.Fatalf("error model deploy %v ", err)
		}
	},
}

func init() {
	deployCmd.Flags().StringVarP(&configFile, "config", "c", "", "the output format to use")

	modelCmd.AddCommand(deployCmd)
}
