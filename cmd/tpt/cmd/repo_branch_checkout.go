// Copyright © 2018 One Concern

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// branchCheckoutCmd represents the checkout command
var branchCheckoutCmd = &cobra.Command{
	Use:   "checkout",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("checkout called")
	},
}

func init() {
	branchCmd.AddCommand(branchCheckoutCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// branchCheckoutCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// branchCheckoutCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
