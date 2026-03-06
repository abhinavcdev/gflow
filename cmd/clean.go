package cmd

import (
	"fmt"
	"strings"

	"github.com/abhinavcdev/gflow/internal/git"
	"github.com/abhinavcdev/gflow/internal/ui"
	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Prune merged and stale branches — local and remote",
	Long: `Remove branches that have been merged or are stale.
Safely deletes local branches that have been merged into the base branch,
and optionally removes their remote tracking branches.

Examples:
  gflow clean                # Interactive cleanup
  gflow clean --merged       # Only remove merged branches
  gflow clean --dry-run      # Preview what would be deleted
  gflow clean --force        # Skip confirmation`,
	RunE: runClean,
}

var (
	cleanMergedOnly bool
	cleanDryRun     bool
	cleanForce      bool
	cleanRemote     bool
)

func init() {
	cleanCmd.Flags().BoolVar(&cleanMergedOnly, "merged", false, "only remove merged branches")
	cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "preview without deleting")
	cleanCmd.Flags().BoolVarP(&cleanForce, "force", "f", false, "skip confirmation")
	cleanCmd.Flags().BoolVar(&cleanRemote, "remote", false, "also delete remote branches")
}

func runClean(cmd *cobra.Command, args []string) error {
	g := git.NewFromCwd()

	if !g.IsRepo() {
		ui.Error("Not a git repository")
		return fmt.Errorf("not a git repository")
	}

	ui.Title("  Clean Branches")
	fmt.Println()

	currentBranch, _ := g.CurrentBranch()

	// Fetch to get latest remote state
	s := ui.StartSpinner("Fetching latest remote state...")
	_ = g.Fetch()
	ui.StopSpinner(s, "Fetched remote state")

	// Get merged branches
	mergedOut, err := g.Run("branch", "--merged", cfg.Branching.Main)
	if err != nil {
		ui.Error("Failed to list merged branches")
		return err
	}

	var toDelete []string
	protected := map[string]bool{
		cfg.Branching.Main:    true,
		cfg.Branching.Develop: true,
		currentBranch:         true,
		"main":                true,
		"master":              true,
		"develop":             true,
		"dev":                 true,
	}

	for _, line := range strings.Split(mergedOut, "\n") {
		branch := strings.TrimSpace(strings.TrimPrefix(line, "* "))
		if branch == "" {
			continue
		}
		if protected[branch] {
			continue
		}
		toDelete = append(toDelete, branch)
	}

	if len(toDelete) == 0 {
		ui.SuccessMsg("No branches to clean up")
		fmt.Println()
		return nil
	}

	// Display branches to delete
	fmt.Printf("  Found %d merged branch(es) to clean:\n\n", len(toDelete))
	for _, b := range toDelete {
		branchType, name := parseBranchName(b)
		icon := "  "
		switch branchType {
		case "feature", "feat":
			icon = "✨"
		case "bugfix", "fix":
			icon = "🐛"
		case "hotfix":
			icon = "🔥"
		case "release":
			icon = "🏷 "
		}
		fmt.Printf("    %s %s %s\n", icon, b, ui.MutedStyle.Render("("+name+")"))
	}
	fmt.Println()

	if cleanDryRun {
		ui.Info("Dry run — no branches were deleted")
		return nil
	}

	// Confirm
	if !cleanForce {
		proceed, err := ui.PromptConfirm(fmt.Sprintf("Delete %d branch(es)?", len(toDelete)), false)
		if err != nil || !proceed {
			ui.Info("Aborted")
			return nil
		}
	}

	fmt.Println()

	// Delete branches
	deleted := 0
	failed := 0
	for _, b := range toDelete {
		if err := g.DeleteBranch(b); err != nil {
			if err := g.ForceDeleteBranch(b); err != nil {
				ui.StepFail(fmt.Sprintf("Failed to delete %s", b))
				failed++
				continue
			}
		}
		ui.StepDone(fmt.Sprintf("Deleted %s", b))
		deleted++

		// Also delete remote if flag is set
		if cleanRemote {
			if err := g.DeleteRemoteBranch(b); err != nil {
				ui.Warn(fmt.Sprintf("Could not delete remote %s", b))
			} else {
				ui.StepDone(fmt.Sprintf("Deleted remote %s", b))
			}
		}
	}

	fmt.Println()
	ui.SuccessMsg(fmt.Sprintf("Cleaned %d branch(es)", deleted))
	if failed > 0 {
		ui.Warn(fmt.Sprintf("%d branch(es) could not be deleted", failed))
	}
	fmt.Println()

	return nil
}
