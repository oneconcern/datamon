// Copyright © 2018 One Concern

package cmd

import (
	"log"

	"github.com/oneconcern/trumpet"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// dumpCmd represents the dump command
var dumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "Print the config used",
	Long:  `Print the config used by the invocation of the tpt command`,
	Run: func(cmd *cobra.Command, args []string) {
		var cfg trumpet.Config
		if err := viper.Unmarshal(&cfg); err != nil {
			log.Fatalln(err)
		}

		print(cfg)
	},
}

func init() {
	configCmd.AddCommand(dumpCmd)
	if err := addFormatFlag(dumpCmd, "yaml"); err != nil {
		log.Fatalln(err)
	}
}
