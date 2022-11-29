package cmd

import (
	"github.com/spf13/cobra"
)

// purgeCmd represents the purge related commands
var purgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Commands to purge unused blob storage",
	Long: `Purge allows owners of a BLOB storage to actually delete data that is no longer referenced by any repo.

To effectively proceed to a purge, proceed with the following steps:
1. Use "datamon repo delete" to delete repositories. This will remove references to a repo. Actual BLOB storage is maintained.
2. Use "datamon purge build-reverse-lookup". This will build an index all currently active BLOB references for _all_ repositories.
3. Use "datamon purge delete-unused". This will delete BLOB resources that are not present in the index.

NOTES:
* datamon purge delete-unused-blobs won't start if no reverse-lookup index is present
* datamon purge build-reverse-lookup may be run again, thus updating the index
* the update time considered for the reverse-lookup index is the time the build command is launched
* any repo or file object that is created while building the index will be ignored in the index.
* when running delete-unused, BLOB pages that are more recent than the index won't be removed.
`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := newCliOptionInputs(config, &datamonFlags).populateRemoteConfig(); err != nil {
			wrapFatalln("populate remote config", err)
		}
	},
}
