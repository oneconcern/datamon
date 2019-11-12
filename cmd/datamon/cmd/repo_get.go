package cmd

import (
	"bytes"
	"context"
	"log"

	"github.com/oneconcern/datamon/pkg/core"
	status "github.com/oneconcern/datamon/pkg/core/status"

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
			wrapFatalln("create remote stores", err)
			return
		}
		repoDescriptor, err := core.GetRepoDescriptorByRepoName(
			remoteStores.meta, params.repo.RepoName)
		if err == status.ErrNotFound {
			wrapFatalWithCode(int(unix.ENOENT), "didn't find repo %q", params.repo.RepoName)
			return
		}
		if err != nil {
			wrapFatalln("error downloading repo information", err)
			return
		}

		var buf bytes.Buffer
		err = repoDescriptorTemplate.Execute(&buf, repoDescriptor)
		if err != nil {
			wrapFatalln("executing template", err)
			return
		}
		log.Println(buf.String())
	},
}

func init() {
	requiredFlags := []string{addRepoNameOptionFlag(GetRepoCommand)}

	for _, flag := range requiredFlags {
		err := GetRepoCommand.MarkFlagRequired(flag)
		if err != nil {
			wrapFatalln("mark required flag", err)
			return
		}
	}

	repoCmd.AddCommand(GetRepoCommand)
}
