package cmd

import (
	"github.com/abhinavcdev/gflow/internal/config"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
	cfg     *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "gflow",
	Short: "gflow — opinionated git workflow CLI",
	Long: `gflow encodes your team's branching + PR conventions into one CLI.

Create branches, commits, push, open PRs with templates, assign reviewers — 
all in one command. Works with GitHub, GitLab, and Bitbucket.

Get started:
  gflow init          Initialize gflow in your repository
  gflow start         Create a new feature/bugfix/hotfix branch
  gflow pr            Create a pull request from current branch
  gflow finish        Merge and clean up a branch
  gflow sync          Sync current branch with upstream
  gflow release       Create a release with version bump and changelog`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if cmd.Name() == "init" || cmd.Name() == "version" || cmd.Name() == "help" {
			return
		}
		cfg = config.LoadOrDefault()
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: .gflow.yml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(prCmd)
	rootCmd.AddCommand(finishCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(releaseCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(checkoutCmd)
	rootCmd.AddCommand(logCmd)
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(commitCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(reopenCmd)
}
