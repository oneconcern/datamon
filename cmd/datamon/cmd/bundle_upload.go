package cmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"text/template"
	"time"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/model"
	"go.uber.org/zap"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const (
	fileUploadsByConcurrencyFactor = 5
)

var uploadTemplate func(flagsT) *template.Template

// uploadBundleCmd is the command to upload a bundle from Datamon and model it locally.
var uploadBundleCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a bundle",
	Long: `Upload a bundle consisting of all files stored in a directory,
to the cloud backend storage.
l.l
This is analogous to the "git commit" command. A message and a label may be set.
`,
	Example: `% datamon bundle upload --path /path/to/data/folder --message "The initial commit for the repo" --repo ritesh-test-repo --label init
Uploading blob:0871e8f83bdefd710a7710de14decef2254ffed94ee537d72eef671fa82d72d10015b3758b0a8960c93899af265191b0108663c95ece8377bf89e741e14f2a53, bytes:1440
Uploaded bundle id:1INzQ5TV4vAAfU2PbRFgPfnzEwR
set label 'init'
`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		defer func(t0 time.Time) {
			cliUsage(t0, "bundle upload", err)
		}(time.Now())

		ctx := context.Background()

		optionInputs := newCliOptionInputs(config, &datamonFlags)
		contributor, err := optionInputs.contributor()
		if err != nil {
			wrapFatalln("populate contributor struct", err)
			return
		}
		sourceStore, err := optionInputs.srcStore(ctx, false)
		if err != nil {
			wrapFatalln("create source store", err)
			return
		}
		bd := model.NewBundleDescriptor(
			model.Message(datamonFlags.bundle.Message),
			model.BundleContributor(contributor),
		)

		bundleOpts, err := optionInputs.bundleOpts(ctx)
		if err != nil {
			wrapFatalln("failed to initialize bundle options", err)
			return
		}
		bundleOpts = append(bundleOpts, core.BundleDescriptor(bd))
		bundleOpts = append(bundleOpts, core.ConsumableStore(sourceStore))
		bundleOpts = append(bundleOpts, core.Repo(datamonFlags.repo.RepoName))
		bundleOpts = append(bundleOpts, core.SkipMissing(datamonFlags.bundle.SkipOnError))
		bundleOpts = append(bundleOpts,
			core.ConcurrentFileUploads(getConcurrencyFactor(fileUploadsByConcurrencyFactor)))
		logger, err := optionInputs.getLogger()
		if err != nil {
			wrapFatalln("get logger", err)
			return
		}
        logger.Debug("RES-10456/gcs-retry-logic", zap.String("bundle.FileList", datamonFlags.bundle.FileList))
		bundleOpts = append(bundleOpts, core.Logger(logger))
		bundleOpts = append(bundleOpts, core.BundleWithMetrics(datamonFlags.root.metrics.IsEnabled()))
		bundleOpts = append(bundleOpts, core.BundleWithRetry(datamonFlags.fs.WithRetry))

		// feature guard
		if enableBundlePreserve {
			bundleOpts = append(bundleOpts, core.BundleID(datamonFlags.bundle.ID))
		}
		bundle := core.NewBundle(
			bundleOpts...,
		)

		if datamonFlags.bundle.FileList != "" {
            logger.Debug("RES-10456/gcs-retry-logic - has filelist trying to upload specific keys")
			getKeys := func() ([]string, error) {
				var file afero.File
				file, err = os.Open(datamonFlags.bundle.FileList)
				if err != nil {
					return nil, fmt.Errorf("failed to open file: %s: %w", datamonFlags.bundle.FileList, err)
				}
				lineScanner := bufio.NewScanner(file)
				files := make([]string, 0)
				for lineScanner.Scan() {
					files = append(files, lineScanner.Text())
				}
				return files, nil
			}
			err = core.UploadSpecificKeys(ctx, bundle, getKeys)
			if err != nil {
				wrapFatalln("upload bundle by filelist", err)
				return
			}
		} else {
            logger.Debug("RES-10456/gcs-retry-logic - does not have filelist, uploading full bundle")
			err = core.Upload(ctx, bundle)
			if err != nil {
				wrapFatalln("upload bundle", err)
				return
			}
		}

		var labelSet string
		defer func() {
			var buf bytes.Buffer
			if ert := uploadTemplate(datamonFlags).Execute(&buf, struct {
				core.Bundle
				Label string
			}{Bundle: *bundle, Label: labelSet}); ert != nil {
				wrapFatalln("executing template", ert)
			}
			log.Println(buf.String())
		}()

		if datamonFlags.label.Name != "" {
			label := core.NewLabel(
				core.LabelWithMetrics(datamonFlags.root.metrics.IsEnabled()),
				core.LabelDescriptor(
					model.NewLabelDescriptor(
						model.LabelContributor(contributor),
						model.LabelName(datamonFlags.label.Name),
					),
				))
			err = label.UploadDescriptor(ctx, bundle)
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
	requireFlags(uploadBundleCmd,
		addRepoNameOptionFlag(uploadBundleCmd),
		addPathFlag(uploadBundleCmd),
		addCommitMessageFlag(uploadBundleCmd),
	)

	addFileListFlag(uploadBundleCmd)
	addLabelNameFlag(uploadBundleCmd)
	addSkipMissingFlag(uploadBundleCmd)
	addConcurrencyFactorFlag(uploadBundleCmd, 100)

	// feature guard
	if enableBundlePreserve {
		addBundleFlag(uploadBundleCmd)
	}

	bundleCmd.AddCommand(uploadBundleCmd)

	uploadTemplate = func(opts flagsT) *template.Template {
		if opts.core.Template != "" {
			t, err := template.New("uploaded").Parse(datamonFlags.core.Template)
			if err != nil {
				wrapFatalln("invalid template", err)
			}
			return t
		}
		const uploadTemplateString = `Uploaded bundle id:{{.BundleID}}
{{- if .Label }}
set label '{{.Label}}'
{{- end }}`
		return template.Must(template.New("uploaded").Parse(uploadTemplateString))
	}
}
