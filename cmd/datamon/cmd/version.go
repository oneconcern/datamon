package cmd

import (
	"bytes"
	"github.com/spf13/cobra"
)

var (
	Version   string
	BuildDate string
	GitCommit string
	GitState  string
)

type VersionInfo struct {
	Version   string `json:"version,omitempty"`
	BuildDate string `json:"buildDate,omitempty"`
	GitCommit string `json:"gitCommit,omitempty"`
	GitState  string `json:"gitState,omitempty"`
}

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

func (v VersionInfo) String() string {
	var buf bytes.Buffer
	buf.WriteString("Version: ")
	buf.WriteString(v.Version)
	buf.WriteString("\n")
	buf.WriteString("Build date: ")
	buf.WriteString(v.BuildDate)
	buf.WriteString("\n")
	buf.WriteString("Commit: ")
	buf.WriteString(v.GitCommit)
	buf.WriteString("\n")
	buf.WriteString("Working tree: ")
	buf.WriteString(v.GitState)
	buf.WriteString("\n")
	return buf.String()
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
		logStdOut(NewVersionInfo().String())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
