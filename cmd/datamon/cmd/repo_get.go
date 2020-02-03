package cmd

import (
	"bytes"
	"context"

	"github.com/oneconcern/datamon/pkg/core"
	status "github.com/oneconcern/datamon/pkg/core/status"
	"github.com/oneconcern/datamon/pkg/errors"

	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

// GetRepoCommand retrieves the description of a repo
var GetRepoCommand = &cobra.Command{
	Use:   "get",
	Short: "Get repo info by name",
	Long: `Performs a direct lookup of repos by name.
Prints corresponding repo information if the name exists,
exits with ENOENT status otherwise.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		remoteStores, err := paramsToDatamonContext(ctx, datamonFlags)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		repoDescriptor, err := core.GetRepoDescriptorByRepoName(
			remoteStores, datamonFlags.repo.RepoName)
		if err != nil {
			if errors.Is(err, status.ErrNotFound) {
				wrapFatalWithCodef(int(unix.ENOENT), "didn't find repo %q", datamonFlags.repo.RepoName)
				return
			}
			wrapFatalln("error downloading repo information", err)
			return
		}

		var buf bytes.Buffer
		err = repoDescriptorTemplate(datamonFlags).Execute(&buf, repoDescriptor)
		if err != nil {
			wrapFatalln("executing template", err)
			return
		}
		log.Println(buf.String())
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		config.populateRemoteConfig(&datamonFlags)
	},
}

func init() {
	requireFlags(GetRepoCommand,
		addRepoNameOptionFlag(GetRepoCommand),
	)

	repoCmd.AddCommand(GetRepoCommand)
}
