package cmd

import "github.com/spf13/cobra"

var dummy string

func addContributorEmail(cmd *cobra.Command) string {
	contributorEmail := "email"
	cmd.Flags().StringVar(&dummy, contributorEmail, "", "The email of the contributor")
	_ = cmd.Flags().MarkDeprecated("email", "now ignored. Using oauth authentication instead")
	return contributorEmail
}

func addContributorName(cmd *cobra.Command) string {
	contributorName := "name"
	cmd.Flags().StringVar(&dummy, contributorName, "", "The name of the contributor")
	_ = cmd.Flags().MarkDeprecated("name", "now ignored. Using oauth authentication instead")
	return contributorName
}
