package cmd

import (
	"bytes"
	"text/template"

	"github.com/spf13/cobra"
)

var (
	// statically linked variables when building releases

	Version   string
	BuildDate string
	GitCommit string
	GitState  string
)

var versionTemplate func(flagsT) *template.Template

// VersionInfo describe versioning information about a build
type VersionInfo struct {
	Version   string `json:"version,omitempty"`
	BuildDate string `json:"buildDate,omitempty"`
	GitCommit string `json:"gitCommit,omitempty"`
	GitState  string `json:"gitState,omitempty"`
}

// NewVersionInfo yields version information about this build
func NewVersionInfo() VersionInfo {
	ver := VersionInfo{
		Version:   "dev",
		BuildDate: BuildDate,
		GitCommit: GitCommit,
		GitState:  "",
	}
	if Version != "" {
		ver.Version = Version
		ver.GitState = "clean"
	}
	if GitState != "" {
		ver.GitState = GitState
	}
	return ver
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "prints the version of datamon",
	Long: `Prints the version of datamon. It includes the following components:
	* Semver (output of git describe --tags)
	* Build Date (date at which the binary was built)
	* Git Commit (the git commit hash this binary was built from
	* Git State (when dirty there were uncommitted changes during the build)
`,
	Run: func(cmd *cobra.Command, args []string) {
		var buf bytes.Buffer
		if err := versionTemplate(datamonFlags).Execute(&buf, NewVersionInfo()); err != nil {
			wrapFatalln("executing template", err)
		}
		log.Println(buf.String())
	},
}

func init() {
	addTemplateFlag(versionCmd)
	rootCmd.AddCommand(versionCmd)

	versionTemplate = func(opts flagsT) *template.Template {
		if opts.core.Template != "" {
			t, err := template.New("version").Parse(datamonFlags.core.Template)
			if err != nil {
				wrapFatalln("invalid template", err)
			}
			return t
		}
		const versionTemplateString = `Version: {{.Version}}
BuildDate: {{.BuildDate}}
Commit: {{.GitCommit}}
Working tree: {{.GitState}}`
		return template.Must(template.New("version").Parse(versionTemplateString))
	}
}
