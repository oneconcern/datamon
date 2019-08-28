// Copyright © 2018 One Concern

package cmd

import (
	"bytes"
	"context"
	"log"
	"text/template"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/storage/gcs"
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
		sourceStore, err := gcs.New(ctx, params.repo.MetadataBucket, config.Credential)
		if err != nil {
			logFatalln(err)
		}
		blobStore, err := gcs.New(ctx, params.repo.BlobBucket, config.Credential)
		if err != nil {
			logFatalln(err)
		}
		path, err := sanitizePath(params.bundle.DataPath)
		if err != nil {
			logFatalln("Failed path validation: " + err.Error())
		}
		fs := afero.NewBasePathFs(afero.NewOsFs(), path+"/")
		destinationStore := localfs.New(fs)

		err = setLatestOrLabelledBundle(ctx, sourceStore)
		if err != nil {
			logFatalln(err)
		}

		localBundle := core.New(core.NewBDescriptor(),
			core.ConsumableStore(destinationStore),
		)
		remoteBundle := core.New(core.NewBDescriptor(),
			core.Repo(params.repo.RepoName),
			core.MetaStore(sourceStore),
			core.BlobStore(blobStore),
			core.BundleID(params.bundle.ID),
			core.ConcurrentFilelistDownloads(
				params.bundle.ConcurrencyFactor/filelistDownloadsByConcurrencyFactor),
		)

		diff, err := core.Diff(ctx, localBundle, remoteBundle)
		if err != nil {
			logFatalln(err)
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
}

func init() {

	// Source
	requiredFlags := []string{addRepoNameOptionFlag(bundleDiffCmd)}

	// Destination
	requiredFlags = append(requiredFlags, addDataPathFlag(bundleDiffCmd))

	// Bundle to download
	addBundleFlag(bundleDiffCmd)
	// Blob bucket
	addBlobBucket(bundleDiffCmd)
	addBucketNameFlag(bundleDiffCmd)

	addLabelNameFlag(bundleDiffCmd)

	addConcurrencyFactorFlag(bundleDiffCmd)

	for _, flag := range requiredFlags {
		err := bundleDiffCmd.MarkFlagRequired(flag)
		if err != nil {
			logFatalln(err)
		}
	}

	bundleCmd.AddCommand(bundleDiffCmd)
}
