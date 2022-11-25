package cmd

import (
	"context"
	"time"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// deleteLookupCmd represents the command to build a reverse-lookup index of BLOB resources.
var deleteLookupCmd = &cobra.Command{
	Use:   "delete-reverse-lookup",
	Short: "Command to delete a reverse-lookup index from the metadata",
	Long: `The index maybe quite large and only really used when we need to purge BLOBs.

This command allows to remove the index file from the metadata.
Only ONE instance of this command may run: dropping index concurrently is not supported.

A deletion of the index may be forced using the "--force" flag.

You MUST make sure that no concurrent build-reverse-lookup or delete job is still running before doing that.
`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := newCliOptionInputs(config, &datamonFlags).populateRemoteConfig(); err != nil {
			wrapFatalln("populate remote config", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		defer func(t0 time.Time) {
			cliUsage(t0, "purge build-reverse-lookup", err)
		}(time.Now())

		ctx := context.Background()
		optionInputs := newCliOptionInputs(config, &datamonFlags)
		remoteStores, err := optionInputs.datamonContext(ctx)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		logger, err := optionInputs.getLogger()

		logger.Info("deleting reverse-lookup index",
			zap.String("context", datamonFlags.context.Descriptor.Name),
			zap.Bool("force?", datamonFlags.purge.Force),
			zap.String("context BLOB bucket", datamonFlags.context.Descriptor.Blob),
			zap.String("context metadata bucket", datamonFlags.context.Descriptor.Metadata),
		)

		err = core.PurgeLock(remoteStores, core.WithPurgeForce(datamonFlags.purge.Force))
		if err != nil {
			wrapFatalln("deleting reverse-lookup: another purge job is running", err)

			return
		}

		err = core.PurgeDropReverseIndex(remoteStores, core.WithPurgeForce(datamonFlags.purge.Force))
		if err != nil {
			erp := core.PurgeUnlock(remoteStores)
			if erp != nil {
				wrapFatalWithCodef(2,
					`building reverse-lookup failed: %v.\n`+
						`Failed to unlock: %v.\n`+
						`Use the '--force' flag on subsequent runs`,
					err, erp,
				)
			}

			wrapFatalln("building reverse-lookup (could remove job lock before exiting)", err)

			return
		}
	},
}
