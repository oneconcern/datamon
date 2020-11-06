package cmd

import (
	"text/template"

	"github.com/spf13/cobra"
)

var (
	useDiamondTemplate        func(flagsT) *template.Template
	diamondDescriptorTemplate func(flagsT) *template.Template
)

// DiamondCmd is the root command for all diamond related subcommands
var DiamondCmd = &cobra.Command{
	Use:   "diamond",
	Short: "Commands to manage diamonds",
	Long:  `A diamond is a parallel data upload operation, which ends up in a single bundle commit.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := newCliOptionInputs(config, &datamonFlags).populateRemoteConfig(); err != nil {
			wrapFatalln("populate remote config", err)
		}
	},
}

func init() {
	addTemplateFlag(DiamondCmd)
	addSkipAuthFlag(DiamondCmd)

	rootCmd.AddCommand(DiamondCmd)
}

func init() {
	useDiamondTemplate = func(opts flagsT) *template.Template {
		const listLineTemplateString = `{{.DiamondID}}`
		return template.Must(template.New("list line").Parse(listLineTemplateString))
	}
	diamondDescriptorTemplate = func(opts flagsT) *template.Template {
		if opts.core.Template != "" {
			t, err := template.New("version").Parse(datamonFlags.core.Template)
			if err != nil {
				wrapFatalln("invalid template", err)
			}
			return t
		}
		const listLineTemplateString = `{{.DiamondID}},started: {{.StartTime}},{{if not .EndTime.IsZero}}ended: {{.EndTime}}{{else}}not terminated{{end}},{{.State}},{{.BundleID}},` +
			`{{.Mode}},{{if .HasConflicts}}hasConflicts{{end}},{{if .HasCheckpoints}}hasCheckpoints{{end}},{{.Tag}}`
		return template.Must(template.New("list line").Parse(listLineTemplateString))
	}
}
