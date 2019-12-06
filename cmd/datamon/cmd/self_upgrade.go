package cmd

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"text/template"

	"syscall"

	"github.com/blang/semver"
	"github.com/oneconcern/datamon/pkg/errors"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	"github.com/spf13/cobra"
)

const (
	githubRepo = "fredbi/datamon"
)

var releaseDescriptorTemplate *template.Template

func init() {
	releaseDescriptorTemplate = func() *template.Template {
		const releaseTemplateString = `
************************************************************
Version: {{ printf "%v" .Version}}
Published on: {{ printf "%v" .PublishedAt }}
Repository: github.com/{{ .RepoOwner }}/{{ .RepoName }}
URL: {{ .URL }}
Release Notes: {{ .ReleaseNotes }}
************************************************************
`
		return template.Must(template.New("release").Parse(releaseTemplateString))
	}()
}

func applyReleaseTemplate(release *selfupdate.Release) error {
	// formats and outputs info about release
	var buf bytes.Buffer
	if err := releaseDescriptorTemplate.Execute(&buf, release); err != nil {
		return errors.New("executing template").Wrap(err)
	}
	log.Println(buf.String())
	return nil
}

type upgradeFlags struct {
	checkOnly   bool
	forceUgrade bool
	verbose     bool
	selfBinary  string // use to mock updated binary (we don't want the test binary to be overwritten during tests)
}

func doSelfUpgrade(opts upgradeFlags) error {
	var err error

	if opts.selfBinary == "" {
		opts.selfBinary, err = os.Executable()
		if err != nil {
			return errors.New("cannot determine current executable").Wrap(err)
		}
	}

	version := NewVersionInfo().Version
	v, err := semver.ParseTolerant(version)
	if err != nil {
		if !opts.forceUgrade {
			return errors.New("you are not running a released version of datamon. Skipping upgrade")
		}
		log.Printf("you are not running a released version of datamon (%v). Forcing upgrade", version)
	}
	if opts.verbose {
		selfupdate.EnableLog()
	}

	latest, err := selfupdate.UpdateCommand(opts.selfBinary, v, githubRepo)
	if err != nil {
		return errors.New("binary update failed").Wrap(err)
	}
	if latest.Version.Equals(v) {
		log.Println("you are running the latest version of datamon", version)
	} else {
		log.Println("successfully updated to version", latest.Version)
		err = applyReleaseTemplate(latest)
		if err != nil {
			return errors.New("cannot render release infos").Wrap(err)
		}
	}
	return nil
}

func doCheckVersion() error {
	isRelease := false
	version := NewVersionInfo().Version
	v, err := semver.ParseTolerant(version)
	if err != nil {
		log.Printf("you are not running a released version of datamon (%v). Checking latest release.", version)
	} else {
		log.Printf("you are running released version %v. Checking latest release.", v)
		isRelease = true
	}

	latest, found, err := selfupdate.DefaultUpdater().DetectLatest(githubRepo)
	if err != nil {
		return errors.New(fmt.Sprintf("could not fetch release from github repo (%s)", githubRepo)).Wrap(err)
	}
	if !found {
		return errors.New(fmt.Sprintf("no matching release from github repo (%s)", githubRepo))
	}

	if isRelease && latest.Version.Equals(v) {
		log.Println("you are running the latest version of datamon", version)
		return nil
	}

	log.Printf("currently running release: %v", version)
	log.Printf("latest available release: %v", latest.Version)
	if err := applyReleaseTemplate(latest); err != nil {
		return errors.New("cannot render release infos").Wrap(err)
	}
	return nil
}

var selfUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrades datamon to the latest release",
	Long:  `Checks for the latest release on github repo then upgrades. By default upgrade is skipped if the current datamon is not a released version`,
	Run: func(cmd *cobra.Command, args []string) {
		datamonFlags.upgrade.verbose = datamonFlags.root.logLevel == "debug"
		if datamonFlags.upgrade.checkOnly {
			if err := doCheckVersion(); err != nil {
				wrapFatalln("error checking latest release", err)
			}
		}
		if err := doSelfUpgrade(datamonFlags.upgrade); err != nil {
			wrapFatalln("error trying to update datamon", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(selfUpgradeCmd)

	addUpgradeCheckOnlyFlag(selfUpgradeCmd)
	addUpgradeForceFlag(selfUpgradeCmd)
}

func doExecAfterUpgrade() error {
	log.Printf("running upgraded version...")
	argsWithoutUpgrade := make([]string, 0, len(os.Args))
	for _, arg := range os.Args {
		if arg != "--"+upgradeFlag {
			argsWithoutUpgrade = append(argsWithoutUpgrade, arg)
		}
	}
	bin, err := os.Executable()
	if err != nil {
		return err
	}
	return syscall.Exec(bin, argsWithoutUpgrade, os.Environ())
}
