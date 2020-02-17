package cmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/model"

	"github.com/spf13/cobra"
)

// SplitAddCmd adds a new split to a diamond and starts uploading
var SplitAddCmd = &cobra.Command{
	Use:   "add",
	Short: "adds a new split and starts uploading",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		datamonFlagsPtr := &datamonFlags
		optionInputs := newCliOptionInputs(config, datamonFlagsPtr)
		remoteStores, err := optionInputs.datamonContext(ctx)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		sourceStore, err := optionInputs.srcStore(ctx, false)
		if err != nil {
			wrapFatalln("create source store", err)
			return
		}
		logger, err := optionInputs.getLogger()
		if err != nil {
			wrapFatalln("get logger", err)
			return
		}
		contributor, err := optionInputs.contributor()
		if err != nil {
			wrapFatalln("populate contributor struct", err)
			return
		}

		// file specifying content to upload
		// TODO(fred): factorize with bundle_upload
		var iterator core.KeyIterator
		if datamonFlags.bundle.FileList != "" {
			file, ero := os.Open(datamonFlags.bundle.FileList)
			if ero != nil {
				wrapFatalln(fmt.Sprintf("failed to open file: %s", datamonFlags.bundle.FileList), ero)
			}
			lineScanner := bufio.NewScanner(file)

			iterator = func(_ string) ([]string, error) {
				files := make([]string, 0)
				for lineScanner.Scan() {
					files = append(files, lineScanner.Text())
				}
				return files, nil
			}
		}

		// regexp based filter on content to upload
		// TODO(fred): factorize with bundle_download
		var filter core.KeyFilter
		if datamonFlags.bundle.NameFilter != "" {
			//	filterRex, erc := regexp.Compile(datamonFlags.bundle.FileList)
			var nameFilterRe *regexp.Regexp
			nameFilterRe, err = regexp.Compile(datamonFlags.bundle.NameFilter)
			if err != nil {
				wrapFatalln(fmt.Sprintf("name filter regexp %s didn't build", datamonFlags.bundle.NameFilter), err)
				return
			}
			filter = nameFilterRe.MatchString
		}

		s := core.NewSplit(datamonFlags.repo.RepoName, datamonFlags.diamond.diamondID, remoteStores,
			core.SplitDescriptor(model.NewSplitDescriptor(
				model.SplitTag(datamonFlags.split.tag),
				model.SplitID(datamonFlags.split.splitID),
				model.SplitContributor(contributor),
			)),
			core.SplitConsumableStore(sourceStore),
			core.SplitSkipMissing(datamonFlags.bundle.SkipOnError),
			core.SplitConcurrentFileUploads(getConcurrencyFactor(fileUploadsByConcurrencyFactor)),
			core.SplitKeyIterator(iterator),
			core.SplitKeyFilter(filter),
			core.SplitLogger(logger),
			core.SplitMustExist(datamonFlags.split.splitID != ""),
		)

		split, err := core.CreateSplit(datamonFlags.repo.RepoName, datamonFlags.diamond.diamondID, remoteStores,
			core.SplitDescriptor(&s.SplitDescriptor),
		)
		if err != nil {
			wrapFatalln("split create", err)
			return
		}

		err = s.Upload()
		if err != nil {
			wrapFatalln("split upload", err)
			return
		}

		var buf bytes.Buffer
		err = useSplitTemplate(datamonFlags).Execute(&buf, split)
		if err != nil {
			wrapFatalln("executing template", err)
		}
		log.Println(buf.String())
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := newCliOptionInputs(config, &datamonFlags).populateRemoteConfig(); err != nil {
			wrapFatalln("populate remote config", err)
		}
	},
}

func init() {
	requireFlags(SplitAddCmd,
		addRepoNameOptionFlag(SplitAddCmd),
		addDiamondFlag(SplitAddCmd),
		addPathFlag(SplitAddCmd),
	)
	addFileListFlag(SplitAddCmd)
	addNameFilterFlag(SplitAddCmd)
	addSkipMissingFlag(SplitAddCmd)
	addConcurrencyFactorFlag(SplitAddCmd, 100)
	addSplitFlag(SplitAddCmd)
	addSplitTagFlag(SplitAddCmd)

	SplitCmd.AddCommand(SplitAddCmd)
}
