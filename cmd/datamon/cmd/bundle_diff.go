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

var bundleDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Diff a downloaded bundle with a remote bundle.",
	Long: "Diff a downloaded bundle with a remote bundle.  " +
		"--destination is a location previously passed to the `bundle download` command.",
	Run: func(cmd *cobra.Command, args []string) {

		const listLineTemplateString = `{{.Type}} , {{.Name}} , {{with .Additional}}{{.Size}} , {{.Hash}}{{end}} , {{with .Existing}}{{.Size}} , {{.Hash}}{{end}}`
		listLineTemplate := template.Must(template.New("list line").Parse(listLineTemplateString))

		ctx := context.Background()

		remoteStores, err := paramsToDatamonContext(ctx, datamonFlags)
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

		bundleOpts := paramsToBundleOpts(remoteStores)
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
			log.Println("empty diff")
		} else {
			for _, de := range diff.Entries {
				var buf bytes.Buffer
				err := listLineTemplate.Execute(&buf, de)
				if err != nil {
					log.Println("executing template:", err)
				}
				log.Println(buf.String())
			}
		}
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		config.populateRemoteConfig(&datamonFlags)
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
