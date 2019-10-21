package cmd

import (
	"os"
	"io/ioutil"
	"fmt"

	"github.com/oneconcern/datamon/pkg/sidecar/param"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var repoCreate = &cobra.Command{
	Use:   "parse",
	Short: "Parse and output config",
	Long: `This is a placeholder as FUSE parsing and output is wip.
The UI likely needs to bifurcate along various directions,
and the option and verb combinations remains to resolve.
`,
// todo: fmt.Errorf() to concat errors throughout
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var inputBuffer []byte
		var fuseParams param.FUSEParams

		inputBuffer, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			terminate(err)
		}

		err = yaml.Unmarshal(inputBuffer, &fuseParams)
		if err != nil {
			terminate(err)
		}

//		fmt.Printf("parsed:\n %v\n", fuseParams)

		envVars, err := param.FUSEParamsToEnvVars(fuseParams)
		if err != nil {
			terminate(err)
		}

		for varName, varVal := range envVars {
			fmt.Printf("export %s='%s'\n", varName, varVal)
		}

	},
}

func init() {
	requiredFlags := []string{}

//	requiredFlags = append(requiredFlags, addRepoDescription(repoCreate))


	for _, flag := range requiredFlags {
		err := repoCreate.MarkFlagRequired(flag)
		if err != nil {
			terminate(err)
		}
	}

	rootCmd.AddCommand(repoCreate)

}
