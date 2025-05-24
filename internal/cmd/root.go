package cmd

import (
	"context"
	"time"

	"github.com/fcjr/git-vibe/internal/version"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "git-vibe",
	Short:   "git-vibe is a git plugin that uses an llm to help with common git tasks",
	Version: version.Version,
}

const commandTimeout = 15 * time.Second

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	cobra.CheckErr(rootCmd.ExecuteContext(ctx))
}

func init() {
	rootCmd.SetVersionTemplate(version.String())

	// Root Flags
	rootCmd.Flags().BoolP("version", "v", false, "Get the version of the git-vibe plugin") // overrides default msg

	// Register Commands
	rootCmd.AddCommand(newVersionCmd().cmd)
}
