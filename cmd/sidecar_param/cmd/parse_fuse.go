package cmd

import (
	"os"
	"io/ioutil"
	"fmt"

	"github.com/oneconcern/datamon/pkg/sidecar/param"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var parseFuse = &cobra.Command{
	Use:   "fuse",
	Short: "Parse and output Filesystem in USErspace sidecar config",
	Long: `Read YAML from stdin,
write shell script for env var initialization to stdout
`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var inputBuffer []byte
		var fuseParams param.FUSEParams
		inputBuffer, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			terminate(fmt.Errorf("read input from stdin: %v", err))
		}
		err = yaml.Unmarshal(inputBuffer, &fuseParams)
		if err != nil {
			terminate(fmt.Errorf("deserialize parameters for sidecar: %v", err))
		}
		envVars, err := param.FUSEParamsToEnvVars(fuseParams)
		if err != nil {
			terminate(fmt.Errorf("serialize parameters as environment variables: %v", err))
		}
		for varName, varVal := range envVars {
			fmt.Printf("export %s='%s'\n", varName, varVal)
		}
	},
}

func init() {
	requiredFlags := []string{}

	for _, flag := range requiredFlags {
		err := parseFuse.MarkFlagRequired(flag)
		if err != nil {
			terminate(err)
		}
	}

	configParse.AddCommand(parseFuse)

}
