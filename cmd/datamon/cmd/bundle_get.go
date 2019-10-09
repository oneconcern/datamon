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

var GetBundleCommand = &cobra.Command{
	Use:   "get",
	Short: "Get bundle info by id",
	Long: `Performs a direct lookup of labels by id.
Prints corresponding bundle information if the label exists,
exits with ENOENT status otherwise.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		remoteStores, err := paramsToRemoteCmdStores(ctx, params)
		if err != nil {
			logFatalln(err)
			return
		}

		err = setLatestOrLabelledBundle(ctx, remoteStores.meta)
		if err == core.ErrNotFound {
			fmt.Fprintf(os.Stderr, "didn't find label %q\n", params.label.Name)
			osExit(int(unix.ENOENT))
			return
		}
		if err != nil {
			logFatalln(err)
			return
		}
		bundle := core.New(core.NewBDescriptor(),
			core.Repo(params.repo.RepoName),
			core.MetaStore(remoteStores.meta),
			core.BundleID(params.bundle.ID),
		)

		err = core.DownloadMetadata(ctx, bundle)
		if err == core.ErrNotFound {
			fmt.Fprintf(os.Stderr, "didn't find bundle '%v'\n", params.bundle.ID)
			osExit(int(unix.ENOENT))
			return
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			logFatalf("error downloading bundle information\n")
		}

		var buf bytes.Buffer
		err = bundleDescriptorTemplate.Execute(&buf, bundle.BundleDescriptor)
		if err != nil {
			log.Println("executing template:", err)
		}
		log.Println(buf.String())
	},
}

func init() {
	requiredFlags := []string{addRepoNameOptionFlag(GetBundleCommand)}

	addBundleFlag(GetBundleCommand)
	addLabelNameFlag(GetBundleCommand)

	for _, flag := range requiredFlags {
		err := GetBundleCommand.MarkFlagRequired(flag)
		if err != nil {
			logFatalln(err)
		}
	}

	bundleCmd.AddCommand(GetBundleCommand)
}
