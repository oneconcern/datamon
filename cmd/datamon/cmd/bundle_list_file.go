package cmd

import (
	"bytes"
	"context"
	"text/template"

	"github.com/oneconcern/datamon/pkg/core"

	"github.com/spf13/cobra"
)

var fileLineTemplate func(flagsT) *template.Template

var bundleFileList = &cobra.Command{
	Use:   "files",
	Short: "List files in a bundle",
	Long: `List all the files in a bundle.

You may use the "--label" flag as an alternate way to specify the bundle to search for.

This is analogous to the git command "git show --pretty="" --name-only {commit-ish}".
`,
	Example: `% datamon bundle list files --repo ritesh-test-repo --bundle 1ISwIzeAR6m3aOVltAsj1kfQaml
Using bundle: 1UZ6kpHe3EBoZUTkKPHSf8s2beh
name:bundle_upload.go, size:4021, hash:b9258e91eb29fe42c70262dd2da46dd71385995dbb989e6091328e6be3d9e3161ad22d9ad0fbfb71410f9e4730f6ac4482cc592c0bc6011585bd9b0f00b11463
...`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		remoteStores, err := paramsToDatamonContext(ctx, datamonFlags)
		if err != nil {
			wrapFatalln("create remote stores", err)
			return
		}
		err = setLatestOrLabelledBundle(ctx, remoteStores)
		if err != nil {
			wrapFatalln("determine bundle id", err)
			return
		}
		bundleOpts := paramsToBundleOpts(remoteStores)
		bundleOpts = append(bundleOpts, core.Repo(datamonFlags.repo.RepoName))
		bundleOpts = append(bundleOpts, core.BundleID(datamonFlags.bundle.ID))
		bundle := core.NewBundle(core.NewBDescriptor(),
			bundleOpts...,
		)
		err = core.PopulateFiles(context.Background(), bundle)
		if err != nil {
			wrapFatalln("download filelist", err)
			return
		}
		for _, e := range bundle.BundleEntries {
			var buf bytes.Buffer
			if err := fileLineTemplate(datamonFlags).Execute(&buf, e); err != nil {
				wrapFatalln("executing template", err)
			}
			log.Println(buf.String())
		}
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		config.populateRemoteConfig(&datamonFlags)
	},
}

func init() {
	requireFlags(bundleFileList,
		// Source
		addRepoNameOptionFlag(bundleFileList),
	)

	// Bundle to download
	addBundleFlag(bundleFileList)

	addLabelNameFlag(bundleFileList)
	addTemplateFlag(bundleFileList)

	BundleListCommand.AddCommand(bundleFileList)

	fileLineTemplate = func(opts flagsT) *template.Template {
		if opts.core.Template != "" {
			t, err := template.New("file line").Parse(datamonFlags.core.Template)
			if err != nil {
				wrapFatalln("invalid template", err)
			}
			return t
		}
		const fileLineTemplateString = `name:{{.NameWithPath}}, size:{{.Size}}, hash:{{.Hash}}`
		return template.Must(template.New("file line").Parse(fileLineTemplateString))
	}
}
