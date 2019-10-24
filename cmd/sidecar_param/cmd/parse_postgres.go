package cmd

import (
	"os"
	"io/ioutil"
	"fmt"

	"github.com/oneconcern/datamon/pkg/sidecar/param"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var parsePostgres = &cobra.Command{
	Use:   "postgres",
	Short: "Parse and output Postgres sidecar config",
	Long: `Read YAML from stdin,
write shell script for env var initialization to stdout
`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var inputBuffer []byte
		var pgParams param.PGParams
		inputBuffer, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			terminate(fmt.Errorf("read input from stdin: %v", err))
		}
		err = yaml.Unmarshal(inputBuffer, &pgParams)
		if err != nil {
			terminate(fmt.Errorf("deserialize parameters for sidecar: %v", err))
		}
		envVars, err := param.PGParamsToEnvVars(pgParams)
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
		err := parsePostgres.MarkFlagRequired(flag)
		if err != nil {
			terminate(err)
		}
	}

	configParse.AddCommand(parsePostgres)

}
