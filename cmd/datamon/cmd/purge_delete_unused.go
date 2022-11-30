package cmd

import (
	"context"
	"time"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// deleteUnusedCmd represents the command to delete BLOB resources that are not present in the reverse-lookup index
var deleteUnusedCmd = &cobra.Command{
	Use:   "delete-unused",
	Short: "Command to delete BLOB resources that are not present in the reverse-lookup index",
	Long: `The reverse-lookup index MUST have been created.

Any BLOB resource that is more recent than the index last update date is kept.

Only ONE instance of this command may run: concurrent deletion is not supported.
Index updates cannot be performed while the deletion is ongoing.

If the delete-unused job fais to complete, it may be run again.

To retry on a failed deletion, use the "--force" flag to bypass the lock.
You MUST make sure that no delete job is still running before doing that.
`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := newCliOptionInputs(config, &datamonFlags).populateRemoteConfig(); err != nil {
			wrapFatalln("populate remote config", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		defer func(t0 time.Time) {
			cliUsage(t0, "purge delete-unused", err)
		}(time.Now())

		ctx := context.Background()
		optionInputs := newCliOptionInputs(config, &datamonFlags)
		remoteStores, err := optionInputs.datamonContext(ctx)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		logger, err := optionInputs.getLogger()

		logger.Info("deleting unused blobs",
			zap.String("context", datamonFlags.context.Descriptor.Name),
			zap.Bool("force?", datamonFlags.purge.Force),
			zap.String("context BLOB bucket", datamonFlags.context.Descriptor.Blob),
			zap.String("context metadata bucket", datamonFlags.context.Descriptor.Metadata),
		)
		opts := []core.PurgeOption{
			core.WithPurgeForce(datamonFlags.purge.Force),
			core.WithPurgeLogger(logger),
			core.WithPurgeLocalStore(datamonFlags.purge.LocalStorePath),
			core.WithPurgeDryRun(datamonFlags.purge.DryRun),
			core.WithPurgeParallel(datamonFlags.bundle.ConcurrencyFactor),
		}

		err = core.PurgeLock(remoteStores, opts...)
		if err != nil {
			wrapFatalln("delete-unused: another purge job is running", err)

			return
		}

		descriptor, err := core.PurgeDeleteUnused(remoteStores, opts...)
		erp := core.PurgeUnlock(remoteStores, opts...)

		if erh := handlePurgeErrors(cmd.Name(), err, erp); erh != nil {
			wrapFatalln(cmd.Name(), erh)

			return
		}

		log.Printf(
			"unused blob keys removed (none is actually removed if this is a dry-run).\n"+
				"Metadata store: %v\n"+
				"Blob store: %s\n"+
				"Index built at: %v\n"+
				"Num blob keys scanned: %d\n"+
				"Num blob keys found in use: %d\n"+
				"Num blob keys found more recent than index: %d\n"+
				"Num blob keys deleted: %d\n"+
				"Num bytes relinquished: %d\n"+
				"Dry-run: %t\n",
			remoteStores.Metadata(),
			remoteStores.Blob(),
			descriptor.IndexTime,
			descriptor.ScannedEntries,
			descriptor.IndexedEntries,
			descriptor.MoreRecentEntries,
			descriptor.DeletedEntries,
			descriptor.DeletedSize,
			descriptor.DryRun,
		)
	},
}
