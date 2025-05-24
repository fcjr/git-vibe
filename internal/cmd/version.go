package cmd

import (
	"fmt"

	"github.com/fcjr/git-vibe/internal/validators"
	"github.com/fcjr/git-vibe/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Args:  validators.NoArgs(),
	Short: "Get the version of the git-vibe plugin",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.String())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
