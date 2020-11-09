package cmd

import (
	"text/template"

	"github.com/spf13/cobra"
)

var (
	useSplitTemplate        func(flagsT) *template.Template
	splitDescriptorTemplate func(flagsT) *template.Template
)

// SplitCmd is the root command for all split related subcommands
var SplitCmd = &cobra.Command{
	Use:   "split",
	Short: "Commands to manage splits",
	Long:  `A split is a part of a diamond, which may be used to upload data concurrently`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := newCliOptionInputs(config, &datamonFlags).populateRemoteConfig(); err != nil {
			wrapFatalln("populate remote config", err)
		}
	},
}

func init() {
	addSkipAuthFlag(SplitCmd)
	DiamondCmd.AddCommand(SplitCmd)
}

func init() {
	useSplitTemplate = func(opts flagsT) *template.Template {
		const listLineTemplateString = `{{.SplitID}}`
		return template.Must(template.New("list line").Parse(listLineTemplateString))
	}
	splitDescriptorTemplate = func(opts flagsT) *template.Template {
		if opts.core.Template != "" {
			t, err := template.New("version").Parse(datamonFlags.core.Template)
			if err != nil {
				wrapFatalln("invalid template", err)
			}
			return t
		}
		const listLineTemplateString = `{{.SplitID}},started: {{.StartTime}},{{if not .EndTime.IsZero}}ended: {{.EndTime}}{{else}}not terminated{{end}},{{.State}},{{.SplitEntriesFileCount}}{{.Tag}}`
		return template.Must(template.New("list line").Parse(listLineTemplateString))
	}
}
