package cmd

import (
	"context"
	"time"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// reverseLookupCmd represents the command to build a reverse-lookup index of BLOB resources.
var reverseLookupCmd = &cobra.Command{
	Use:   "build-reverse-lookup",
	Short: "Command to build a reverse-lookup index of used BLOB resources",
	Long: `The index may be updated, unless a delete-unused command is currently running.

Only ONE instance of this command may run: concurrent index building is not supported.

If a build-reverse-lookup OR delete-unused command was running and failed, an update of the index may be forced using the "--force" flag.

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

		logger.Info("building reverse-lookup index",
			zap.String("context", datamonFlags.context.Descriptor.Name),
			zap.Bool("force?", datamonFlags.purge.Force),
			zap.String("context BLOB bucket", datamonFlags.context.Descriptor.Blob),
			zap.String("context metadata bucket", datamonFlags.context.Descriptor.Metadata),
		)

		err = core.PurgeLock(remoteStores, core.WithPurgeForce(datamonFlags.purge.Force))
		if err != nil {
			wrapFatalln("building reverse-lookup: another purge job is running", err)

			return
		}

		err = core.PurgeBuildReverseIndex(remoteStores, core.WithPurgeForce(datamonFlags.purge.Force))
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
