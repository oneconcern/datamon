package kubeless

import (
	kubelessApi "github.com/kubeless/kubeless/pkg/apis/kubeless/v1beta1"
	kubelessUtils "github.com/kubeless/kubeless/pkg/utils"
	"github.com/oneconcern/datamon/pkg/config"
	"log"
)

func DeployFunction(processor config.Processor, bucketUrl string) {
	defaultFunctionSpec := kubelessApi.Function{}
	defaultFunctionSpec.ObjectMeta.Labels = map[string]string{
		"created-by": "kubeless",
		"function":   processor.Name,
	}

	f, err:= getFunctionDescription(processor.Name, "default", processor.Command, bucketUrl+"?raw=true", processor.Dep, processor.Runtime,
		"", "", "", "180", "", processor.Port, false, make([]string, 0),
		make([]string, 0), make([]string, 0), defaultFunctionSpec)

	if err != nil {
		log.Fatalf("Error while creating function %v ", err)
	}

	kubelessClient, err := kubelessUtils.GetKubelessClientOutCluster()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Deploying function...")
	err = kubelessUtils.CreateFunctionCustomResource(kubelessClient, f)
	if err != nil {
		log.Fatalf("Failed to deploy %s. Received:\n%s", processor.Name, err)
	}
	log.Fatalf("Function %s submitted for deployment", processor.Name)
	log.Printf("Check the deployment status executing 'kubeless function ls %s'", processor.Name)

}
