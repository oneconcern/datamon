// Copyright © 2018 One Concern

package cmd

import (
	"github.com/spf13/cobra"
)

// branchCmd represents the branches command
var branchCmd = &cobra.Command{
	Use:   "branch",
	Short: "Commands to manage branches in a data repository.",
	Long: `Commands to manage branches in a data repository.

A branch is a named pointer to a history.
This can be a completely new line of history or it can start from a common ancestor.

This means that the head commit of a branch is always the version dependent repositories see.
	`,
}

func init() {
	repoCmd.AddCommand(branchCmd)
}
