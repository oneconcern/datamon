package cmd

import (
	"bytes"
	"context"
	"fmt"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/model"

	"github.com/spf13/cobra"
)

// CommitDiamondCmd commits a diamond
var CommitDiamondCmd = &cobra.Command{
	Use:   "commit",
	Short: "Commits a diamond",
	Long:  `Commits a diamond to create a bundle, with conflicts handling`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		datamonFlagsPtr := &datamonFlags
		optionInputs := newCliOptionInputs(config, datamonFlagsPtr)
		remoteStores, err := optionInputs.datamonContext(ctx)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		logger, err := optionInputs.getLogger()
		if err != nil {
			wrapFatalln("get logger", err)
			return
		}

		diamond, err := core.GetDiamond(datamonFlags.repo.RepoName, datamonFlags.diamond.diamondID, remoteStores)
		if err != nil {
			wrapFatalln("error retrieving diamond", err)
			return
		}

		mode, err := checkConflictModes(datamonFlags)
		if err != nil {
			wrapFatalln("incompatible conflict flags", err)
		}

		d := core.NewDiamond(datamonFlags.repo.RepoName, remoteStores,
			core.DiamondDescriptor(model.NewDiamondDescriptor(
				model.DiamondClone(diamond),
				model.DiamondTag(datamonFlags.diamond.tag),
				model.DiamondMode(mode),
			)),
			core.DiamondMessage(datamonFlags.bundle.Message),
			core.DiamondConcurrentFileUploads(getConcurrencyFactor(fileUploadsByConcurrencyFactor)),
			core.DiamondLogger(logger),
			core.DiamondBundleID(datamonFlags.bundle.ID),
		)

		err = d.Commit()
		if err != nil {
			wrapFatalln("diamond commit", err)
		}

		// set label
		// TODO(fred): factorize with bundle upload
		var labelSet string
		defer func() {
			var buf bytes.Buffer
			if ert := uploadTemplate(datamonFlags).Execute(&buf, struct {
				core.Bundle
				Label string
			}{Bundle: *d.Bundle, Label: labelSet}); ert != nil {
				wrapFatalln("executing template", ert)
			}
			log.Println(buf.String())
		}()

		if datamonFlags.label.Name != "" {
			label := core.NewLabel(
				core.NewLabelDescriptor(
					core.LabelContributors(d.BundleDescriptor.Contributors),
				),
				core.LabelName(datamonFlags.label.Name),
			)
			err = label.UploadDescriptor(ctx, d.Bundle)
			if err != nil {
				wrapFatalln("upload label", err)
				return
			}
			labelSet = datamonFlags.label.Name
		}
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := newCliOptionInputs(config, &datamonFlags).populateRemoteConfig(); err != nil {
			wrapFatalln("populate remote config", err)
		}
	},
}

func init() {
	requireFlags(CommitDiamondCmd,
		addRepoNameOptionFlag(CommitDiamondCmd),
		addDiamondFlag(CommitDiamondCmd),
		addCommitMessageFlag(CommitDiamondCmd),
	)
	addWithConflictsFlag(CommitDiamondCmd)
	addWithCheckpointFlag(CommitDiamondCmd)
	addNoConflictsFlag(CommitDiamondCmd)
	addIgnoreConflictsFlag(CommitDiamondCmd)

	addDiamondTagFlag(CommitDiamondCmd)

	addLabelNameFlag(CommitDiamondCmd)
	addConcurrencyFactorFlag(CommitDiamondCmd, 100)

	// feature guard
	if enableBundlePreserve {
		addBundleFlag(CommitDiamondCmd)
	}

	DiamondCmd.AddCommand(CommitDiamondCmd)
}

func checkConflictModes(f flagsT) (model.ConflictMode, error) {
	d := f.diamond
	d.withConflicts = !(d.withCheckpoints || d.noConflicts || d.ignoreConflicts)
	incompatible := false

	switch {
	case d.withCheckpoints && (d.ignoreConflicts || d.noConflicts):
		incompatible = true
	case d.noConflicts && d.ignoreConflicts:
		incompatible = true
	}
	if incompatible {
		return model.ConflictMode(""), fmt.Errorf("cannot specify joint conflict flags")
	}

	switch {
	case d.withCheckpoints:
		return model.EnableCheckpoints, nil
	case d.noConflicts:
		return model.ForbidConflicts, nil
	case d.ignoreConflicts:
		return model.IgnoreConflicts, nil
	default:
		return model.EnableConflicts, nil
	}
}
