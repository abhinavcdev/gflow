package cmd

import (
	"fmt"

	"github.com/abhinavcdev/gflow/internal/git"
	"github.com/abhinavcdev/gflow/internal/provider"
	"github.com/abhinavcdev/gflow/internal/ui"
	"github.com/spf13/cobra"
)

var reopenCmd = &cobra.Command{
	Use:   "reopen [pr-number]",
	Short: "Mark a draft PR as ready for review, or reopen a closed PR",
	Long: `Convert a draft pull request to ready-for-review, or reopen
a previously closed pull request. If no PR number is given, operates 
on the PR associated with the current branch.

Examples:
  gflow reopen             # Ready/reopen the PR for current branch
  gflow reopen 42          # Ready/reopen PR #42
  gflow reopen --ready     # Mark draft as ready (explicit)`,
	RunE: runReopen,
}

var (
	reopenReady bool
)

func init() {
	reopenCmd.Flags().BoolVar(&reopenReady, "ready", false, "explicitly mark draft as ready for review")
}

func runReopen(cmd *cobra.Command, args []string) error {
	g := git.NewFromCwd()

	if !g.IsRepo() {
		ui.Error("Not a git repository")
		return fmt.Errorf("not a git repository")
	}

	ui.Title("  Reopen / Ready PR")
	fmt.Println()

	p, err := provider.New(cfg)
	if err != nil {
		ui.Error("Failed to connect to provider")
		return err
	}

	var pr *provider.PullRequest

	if len(args) > 0 {
		// PR number provided
		var num int
		if _, err := fmt.Sscanf(args[0], "%d", &num); err != nil {
			ui.Errorf("Invalid PR number: %s", args[0])
			return err
		}
		s := ui.StartSpinner(fmt.Sprintf("Fetching PR #%d...", num))
		pr, err = p.GetPR(num)
		if err != nil {
			ui.StopSpinnerFail(s, fmt.Sprintf("Failed to fetch PR #%d", num))
			return err
		}
		ui.StopSpinner(s, fmt.Sprintf("Found PR #%d: %s", pr.Number, pr.Title))
	} else {
		// Find PR for current branch
		branch, err := g.CurrentBranch()
		if err != nil {
			ui.Error("Failed to determine current branch")
			return err
		}

		s := ui.StartSpinner(fmt.Sprintf("Finding PR for %s...", branch))
		pr, err = p.GetPRForBranch(branch)
		if err != nil {
			ui.StopSpinnerFail(s, "No PR found for current branch")
			return err
		}
		ui.StopSpinner(s, fmt.Sprintf("Found PR #%d: %s", pr.Number, pr.Title))
	}

	ui.Detail("State", pr.State)
	ui.Detail("Draft", fmt.Sprintf("%v", pr.Draft))
	fmt.Println()

	// Determine action based on PR state
	switch {
	case pr.Draft || reopenReady:
		// Mark as ready for review — GitHub-specific PATCH
		s := ui.StartSpinner("Marking as ready for review...")
		// Use a generic update approach via the provider
		// For GitHub, we update the draft field
		ghProvider, ok := p.(*provider.GitHub)
		if !ok {
			ui.StopSpinnerFail(s, "Ready-for-review is currently supported on GitHub only")
			ui.Info("Please mark the PR as ready manually on your provider's web UI")
			ui.Detail("URL", pr.URL)
			return nil
		}

		err := ghProvider.MarkReady(pr.Number)
		if err != nil {
			ui.StopSpinnerFail(s, "Failed to mark as ready")
			return err
		}
		ui.StopSpinner(s, "Marked as ready for review!")

	case pr.State == "closed":
		// Reopen the PR
		s := ui.StartSpinner(fmt.Sprintf("Reopening PR #%d...", pr.Number))
		ghProvider, ok := p.(*provider.GitHub)
		if !ok {
			ui.StopSpinnerFail(s, "Reopen is currently supported on GitHub only")
			ui.Info("Please reopen the PR manually on your provider's web UI")
			ui.Detail("URL", pr.URL)
			return nil
		}

		err := ghProvider.ReopenPR(pr.Number)
		if err != nil {
			ui.StopSpinnerFail(s, "Failed to reopen PR")
			return err
		}
		ui.StopSpinner(s, fmt.Sprintf("Reopened PR #%d", pr.Number))

	default:
		ui.Info(fmt.Sprintf("PR #%d is already open and not a draft", pr.Number))
		ui.Detail("URL", pr.URL)
	}

	fmt.Println()
	ui.Detail("URL", pr.URL)
	fmt.Println()

	return nil
}
