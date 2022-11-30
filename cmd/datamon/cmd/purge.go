package cmd

import (
	"context"
	"fmt"

	context2 "github.com/oneconcern/datamon/pkg/context"
	gcscontext "github.com/oneconcern/datamon/pkg/context/gcs"
	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/storage"
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

func handlePurgeErrors(cmdName string, err, erp error) error {
	switch {
	case err != nil && erp != nil:
		return fmt.Errorf(`%v: %v.\n`+
			`Failed to unlock: %v.\n`+
			`Use the '--force' flag on subsequent runs`,
			cmdName, err, erp,
		)

	case err != nil && erp == nil:
		return fmt.Errorf("%v failed (could remove job lock before exiting): %v",
			cmdName, err,
		)

	case err == nil && erp != nil:
		return fmt.Errorf(
			`%v was successful.\n`+
				`But failed to unlock: %v.\n`+
				`Use the '--force' flag on subsequent runs`,
			cmdName, erp,
		)
	default:
		return nil
	}
}

// returns all metadata stores known to the different contexts that share the same blob store
// as the current context.
func metaForSharedContexts(currentContext string, blob storage.Store) ([]context2.Stores, error) {
	configStore := mustGetConfigStore()
	ctx := context.Background()
	contexts, err := core.ListContexts(configStore)
	if err != nil {
		return nil, err
	}

	result := make([]context2.Stores, 0, len(contexts)-1)

	for _, contextName := range contexts {
		if contextName == currentContext {
			continue
		}

		datamonContext, err := context2.GetContext(ctx, configStore, contextName)
		if err != nil {
			return nil, err
		}

		optionInputs := newCliOptionInputs(config, &datamonFlags)
		contextStore, err := gcscontext.MakeContext(ctx, *datamonContext, optionInputs.config.Credential)
		if err != nil {
			return nil, err
		}

		if contextStore.Blob().String() == blob.String() {
			// shared
			result = append(result, contextStore)
		}
	}

	return result, nil
}
