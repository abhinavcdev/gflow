package cmd

import (
	"fmt"

	"github.com/abhinavcdev/gflow/internal/git"
	"github.com/abhinavcdev/gflow/internal/ui"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync current branch with upstream base branch",
	Long: `Fetches latest changes and rebases your branch on top of the base branch.
Keeps your branch up-to-date without merge commits.

Examples:
  gflow sync              # Rebase on default base
  gflow sync --base main  # Rebase on specific branch
  gflow sync --merge      # Merge instead of rebase
  gflow sync --all        # Sync all local branches`,
	RunE: runSync,
}

var (
	syncBase  string
	syncMerge bool
	syncAll   bool
)

func init() {
	syncCmd.Flags().StringVarP(&syncBase, "base", "b", "", "base branch to sync with")
	syncCmd.Flags().BoolVar(&syncMerge, "merge", false, "merge instead of rebase")
	syncCmd.Flags().BoolVar(&syncAll, "all", false, "sync all local branches")
}

func runSync(cmd *cobra.Command, args []string) error {
	g := git.NewFromCwd()

	if !g.IsRepo() {
		ui.Error("Not a git repository")
		return fmt.Errorf("not a git repository")
	}

	ui.Title("  Sync Branch")
	fmt.Println()

	currentBranch, err := g.CurrentBranch()
	if err != nil {
		ui.Error("Failed to determine current branch")
		return err
	}

	// Check for uncommitted changes
	if g.HasChanges() {
		ui.Warn("Uncommitted changes detected — stashing first")
		if err := g.Stash("gflow sync: auto-stash"); err != nil {
			ui.StepFail("Failed to stash changes")
			return err
		}
		defer func() {
			ui.Info("Restoring stashed changes...")
			if err := g.StashPop(); err != nil {
				ui.Warn("Failed to restore stash — run 'git stash pop' manually")
			}
		}()
		ui.StepDone("Stashed changes")
	}

	// Fetch
	s := ui.StartSpinner("Fetching latest from remote...")
	if err := g.Fetch(); err != nil {
		ui.StopSpinnerFail(s, "Failed to fetch")
		return err
	}
	ui.StopSpinner(s, "Fetched latest from remote")

	// Determine base branch
	baseBranch := syncBase
	if baseBranch == "" {
		baseBranch = detectBaseBranch(currentBranch)
	}

	// Update base branch
	s = ui.StartSpinner(fmt.Sprintf("Updating %s...", baseBranch))
	if err := g.Checkout(baseBranch); err != nil {
		ui.StopSpinnerFail(s, fmt.Sprintf("Failed to checkout %s", baseBranch))
		return err
	}
	if err := g.PullRebase(); err != nil {
		ui.StopSpinnerFail(s, fmt.Sprintf("Failed to update %s", baseBranch))
		// Switch back
		_ = g.Checkout(currentBranch)
		return err
	}
	ui.StopSpinner(s, fmt.Sprintf("Updated %s", baseBranch))

	// Switch back to feature branch
	if err := g.Checkout(currentBranch); err != nil {
		ui.StepFail(fmt.Sprintf("Failed to switch back to %s", currentBranch))
		return err
	}

	// Rebase or merge
	if syncMerge {
		s = ui.StartSpinner(fmt.Sprintf("Merging %s into %s...", baseBranch, currentBranch))
		if err := g.Merge(baseBranch, true); err != nil {
			ui.StopSpinnerFail(s, "Merge conflict detected")
			ui.Warn("Resolve conflicts, then run 'git merge --continue'")
			return err
		}
		ui.StopSpinner(s, fmt.Sprintf("Merged %s into %s", baseBranch, currentBranch))
	} else {
		s = ui.StartSpinner(fmt.Sprintf("Rebasing %s onto %s...", currentBranch, baseBranch))
		if err := g.Rebase(baseBranch); err != nil {
			ui.StopSpinnerFail(s, "Rebase conflict detected")
			ui.Warn("Resolve conflicts, then run 'git rebase --continue'")
			ui.Info("Or abort with 'git rebase --abort'")
			return err
		}
		ui.StopSpinner(s, fmt.Sprintf("Rebased onto %s", baseBranch))
	}

	// Show status
	commitCount, err := g.CommitCount(baseBranch)
	if err == nil {
		ui.Detail("Commits ahead", fmt.Sprintf("%d", commitCount))
	}

	fmt.Println()
	ui.SuccessMsg(fmt.Sprintf("Branch %s is up to date with %s", currentBranch, baseBranch))
	fmt.Println()

	return nil
}
