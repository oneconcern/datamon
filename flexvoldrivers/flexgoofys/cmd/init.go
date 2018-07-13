// Copyright © 2018 One Concern

package cmd

import (
	"log"
	"os"

	"github.com/json-iterator/go"

	"github.com/spf13/cobra"
)

type driverStatus string

const (
	dsSuccess      driverStatus = "Success"
	dsFailure      driverStatus = "Failure"
	dsNotSupported driverStatus = "Not supported"
)

func withoutAttach() *driverCapabilities {
	return &driverCapabilities{Attach: false}
}

type driverCapabilities struct {
	Attach bool `json:"attach" yaml:"attach"`
}

type driverOutput struct {
	Status       driverStatus        `json:"status" yaml:"status"`
	Message      string              `json:"message" yaml:"message"`
	Device       string              `json:"device,omitempty" yaml:"device,omitempty"`
	VolumeName   string              `json:"volumeName,omitempty" yaml:"volumeName,omitempty"`
	Attached     *bool               `json:"attached,omitempty" yaml:"attached,omitempty"`
	Capabilities *driverCapabilities `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
}

func respond(status driverStatus, msg string) {
	var data driverOutput
	data.Status = status
	data.Message = msg

	enc := jsoniter.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		log.Fatalln(err)
	}
	if status == dsFailure {
		os.Exit(1)
	}
}

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initializes the driver",
	Long: `Initializes the driver. Called during Kubelet & Controller manager initialization.
On success, the function returns a capabilities map showing whether each Flexvolume capability is supported by the driver.
Current capabilities:

	attach - a boolean field indicating whether the driver requires attach and detach operations.
	This field is required, although for backward-compatibility the default value is set to true,
	i.e. requires attach and detach.
`,
	Run: func(cmd *cobra.Command, args []string) {
		var data driverOutput
		data.Status = dsSuccess
		data.Message = "No initialization required"
		data.Capabilities = withoutAttach()

		enc := jsoniter.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(data); err != nil {
			log.Fatalln(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
