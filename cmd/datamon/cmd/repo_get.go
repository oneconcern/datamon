package cmd

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/oneconcern/datamon/pkg/core"

	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

var GetRepoCommand = &cobra.Command{
	Use:   "get",
	Short: "Get repo info by name",
	Long: `Performs a direct lookup of repos by name.
Prints corresponding repo information if the name exists,
exits with ENOENT status otherwise.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		remoteStores, err := paramsToRemoteCmdStores(ctx, params)
		if err != nil {
			logFatalln(err)
		}
		repoDescriptor, err := core.GetRepoDescriptorByRepoName(
			remoteStores.meta, params.repo.RepoName)
		if err == core.ErrNotFound {
			fmt.Fprintf(os.Stderr, "didn't find repo '%v'\n", params.repo.RepoName)
			osExit(int(unix.ENOENT))
			return
		}
		if err != nil {
			logFatalf("error downloading repo information: %v\n", err)
		}

		var buf bytes.Buffer
		err = repoDescriptorTemplate.Execute(&buf, repoDescriptor)
		if err != nil {
			log.Println("executing template:", err)
		}
		log.Println(buf.String())
	},
}

func init() {
	requiredFlags := []string{addRepoNameOptionFlag(GetRepoCommand)}

	for _, flag := range requiredFlags {
		err := GetRepoCommand.MarkFlagRequired(flag)
		if err != nil {
			logFatalln(err)
		}
	}

	repoCmd.AddCommand(GetRepoCommand)
}
