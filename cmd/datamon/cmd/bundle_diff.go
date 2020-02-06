// Copyright © 2018 One Concern

package cmd

import (
	"bytes"
	"context"
	"text/template"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/storage/localfs"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var bundleDiffTemplate func(flagsT) *template.Template

func init() {
	bundleDiffTemplate = func(opts flagsT) *template.Template {
		if opts.core.Template != "" {
			t, err := template.New("list line").Parse(datamonFlags.core.Template)
			if err != nil {
				wrapFatalln("invalid template", err)
			}
			return t
		}
		const listLineTemplateString = `{{.Type}} , {{.Name}} , {{with .Additional}}{{.Size}} , {{.Hash}}{{end}} , {{with .Existing}}{{.Size}} , {{.Hash}}{{end}}`
		return template.Must(template.New("list line").Parse(listLineTemplateString))
	}
}

var bundleDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Diff a downloaded bundle with a remote bundle.",
	Long: "Diff a downloaded bundle with a remote bundle.  " +
		"--destination is a location previously passed to the `bundle download` command.",
	Run: func(cmd *cobra.Command, args []string) {

		ctx := context.Background()

		datamonFlagsPtr := &datamonFlags
		optionInputs := newCliOptionInputs(config, datamonFlagsPtr)
		remoteStores, err := optionInputs.datamonContext(ctx)
		if err != nil {
			wrapFatalln("failed to initialize remote stores", err)
		}
		path, err := sanitizePath(datamonFlags.bundle.DataPath)
		if err != nil {
			wrapFatalln("failed path validation", err)
			return
		}
		fs := afero.NewBasePathFs(afero.NewOsFs(), path+"/")
		destinationStore := localfs.New(fs)

		err = setLatestOrLabelledBundle(ctx, remoteStores)
		if err != nil {
			wrapFatalln("determine bundle id", err)
			return
		}

		localBundle := core.NewBundle(core.NewBDescriptor(),
			core.ConsumableStore(destinationStore),
		)

		bundleOpts, err := optionInputs.bundleOpts(ctx)
		if err != nil {
			wrapFatalln("failed to initialize bundle options", err)
		}
		bundleOpts = append(bundleOpts, core.Repo(datamonFlags.repo.RepoName))
		bundleOpts = append(bundleOpts, core.BundleID(datamonFlags.bundle.ID))
		bundleOpts = append(bundleOpts,
			core.ConcurrentFilelistDownloads(datamonFlags.bundle.ConcurrencyFactor/filelistDownloadsByConcurrencyFactor))

		remoteBundle := core.NewBundle(core.NewBDescriptor(),
			bundleOpts...,
		)

		diff, err := core.Diff(ctx, localBundle, remoteBundle)
		if err != nil {
			wrapFatalln("bundle diff", err)
			return
		}

		if len(diff.Entries) == 0 {
			// sending this out to stderr (<= no result)
			infoLogger.Println("empty diff")
		} else {
			for _, de := range diff.Entries {
				var buf bytes.Buffer
				err := bundleDiffTemplate(datamonFlags).Execute(&buf, de)
				if err != nil {
					wrapFatalln("executing template:", err)
					return
				}
				log.Println(buf.String())
			}
			// TODO(fred): should probably return some non-zero exit code, like the ordinary diff command
		}
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := newCliOptionInputs(config, &datamonFlags).populateRemoteConfig(); err != nil {
			wrapFatalln("populate remote config", err)
		}
	},
}

func init() {

	requireFlags(bundleDiffCmd,
		addRepoNameOptionFlag(bundleDiffCmd),
		// Destination
		addDataPathFlag(bundleDiffCmd),
	)

	// Bundle to download
	addBundleFlag(bundleDiffCmd)

	addLabelNameFlag(bundleDiffCmd)
	addConcurrencyFactorFlag(bundleDiffCmd, 100)

	bundleCmd.AddCommand(bundleDiffCmd)
}
