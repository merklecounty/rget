package github

import "github.com/spf13/cobra"

func AddCommands(root *cobra.Command) {
	root.AddCommand(generateReleaseSumsCmd)
	root.AddCommand(publishReleaseSumsCmd)
}
