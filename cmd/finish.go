package cmd

import (
	"fmt"

	"github.com/abhinavcdev/gflow/internal/git"
	"github.com/abhinavcdev/gflow/internal/provider"
	"github.com/abhinavcdev/gflow/internal/ui"
	"github.com/spf13/cobra"
)

var finishCmd = &cobra.Command{
	Use:   "finish [branch]",
	Short: "Finish a branch — merge PR, delete branch, clean up",
	Long: `Merges the pull request for the current (or specified) branch, 
deletes the local and remote branch, and switches back to the base branch.

Examples:
  gflow finish                    # Finish current branch
  gflow finish feature/my-thing   # Finish a specific branch
  gflow finish --no-delete        # Keep the branch after merge
  gflow finish --method squash    # Override merge method`,
	RunE: runFinish,
}

var (
	finishMethod   string
	finishNoDelete bool
	finishForce    bool
)

func init() {
	finishCmd.Flags().StringVar(&finishMethod, "method", "", "merge method (merge, squash, rebase)")
	finishCmd.Flags().BoolVar(&finishNoDelete, "no-delete", false, "don't delete branch after merge")
	finishCmd.Flags().BoolVar(&finishForce, "force", false, "force finish even without PR")
}

func runFinish(cmd *cobra.Command, args []string) error {
	g := git.NewFromCwd()

	if !g.IsRepo() {
		ui.Error("Not a git repository")
		return fmt.Errorf("not a git repository")
	}

	ui.Title("  Finish Branch")
	fmt.Println()

	// Determine which branch to finish
	var branch string
	if len(args) > 0 {
		branch = args[0]
	} else {
		var err error
		branch, err = g.CurrentBranch()
		if err != nil {
			ui.Error("Failed to determine current branch")
			return err
		}
	}

	// Don't allow finishing main/develop
	if branch == cfg.Branching.Main || branch == cfg.Branching.Develop {
		ui.Errorf("Cannot finish %s — it's a protected branch", branch)
		return fmt.Errorf("cannot finish %s", branch)
	}

	baseBranch := detectBaseBranch(branch)

	ui.Detail("Branch", branch)
	ui.Detail("Base", baseBranch)
	fmt.Println()

	// Check for uncommitted changes
	if g.HasChanges() {
		ui.Warn("You have uncommitted changes")
		stash, err := ui.PromptConfirm("Stash changes before finishing?", true)
		if err != nil {
			return err
		}
		if stash {
			if err := g.Stash("gflow finish: auto-stash"); err != nil {
				ui.StepFail("Failed to stash changes")
				return err
			}
			ui.StepDone("Stashed changes")
		} else {
			ui.Error("Please commit or stash your changes first")
			return fmt.Errorf("uncommitted changes")
		}
	}

	// Try to find and merge the PR
	mergeMethod := cfg.PR.MergeMethod
	if finishMethod != "" {
		mergeMethod = finishMethod
	}

	p, err := provider.New(cfg)
	if err != nil && !finishForce {
		ui.Warn("Cannot connect to provider — use --force to finish locally")
		return err
	}

	if p != nil {
		// Find the PR for this branch
		s := ui.StartSpinner("Looking for pull request...")
		pr, err := p.GetPRForBranch(branch)
		if err != nil {
			ui.StopSpinnerFail(s, "No open PR found for this branch")
			if !finishForce {
				ui.Info("Use --force to finish without a PR")
				return err
			}
			ui.Info("Proceeding with local merge (--force)")
		} else {
			ui.StopSpinner(s, fmt.Sprintf("Found PR #%d: %s", pr.Number, pr.Title))

			// Check CI status before merging
			s = ui.StartSpinner("Checking CI status...")
			checks, checkErr := p.GetChecks(branch)
			if checkErr != nil {
				ui.StopSpinner(s, "Could not fetch CI status (continuing)")
			} else if len(checks) == 0 {
				ui.StopSpinner(s, "No CI checks configured")
			} else {
				passing, failing, pending := 0, 0, 0
				for _, c := range checks {
					switch c.Status {
					case "success", "neutral", "skipped":
						passing++
					case "failure", "error", "cancelled", "timed_out", "action_required":
						failing++
					default:
						pending++
					}
				}

				if failing > 0 {
					ui.StopSpinnerFail(s, fmt.Sprintf("%d/%d checks failed", failing, len(checks)))
				} else if pending > 0 {
					ui.StopSpinner(s, fmt.Sprintf("%d/%d checks passed, %d pending", passing, len(checks), pending))
				} else {
					ui.StopSpinner(s, fmt.Sprintf("%d/%d checks passed", passing, len(checks)))
				}

				// Show individual checks
				for _, c := range checks {
					var icon string
					switch c.Status {
					case "success":
						icon = ui.SuccessStyle.Render("✓")
					case "failure", "error", "cancelled", "timed_out", "action_required":
						icon = ui.ErrorStyle.Render("✗")
					case "neutral", "skipped":
						icon = ui.MutedStyle.Render("○")
					default:
						icon = ui.WarningStyle.Render("◌")
					}
					fmt.Printf("    %s %s\n", icon, c.Name)
				}

				// Block merge if checks are failing
				if failing > 0 && !finishForce {
					fmt.Println()
					ui.Error("CI checks must pass before merging")
					ui.Info("Use --force to merge anyway")
					return fmt.Errorf("%d CI checks failed", failing)
				}

				if pending > 0 && !finishForce {
					fmt.Println()
					proceed, promptErr := ui.PromptConfirm("Some checks are still pending. Merge anyway?", false)
					if promptErr != nil || !proceed {
						ui.Info("Wait for checks to complete, then run gflow finish again")
						return fmt.Errorf("checks pending")
					}
				}
			}

			fmt.Println()

			// Merge the PR
			s = ui.StartSpinner(fmt.Sprintf("Merging PR #%d (%s)...", pr.Number, mergeMethod))
			err = p.MergePR(pr.Number, provider.PRMergeOptions{
				Method:       mergeMethod,
				DeleteBranch: !finishNoDelete,
			})
			if err != nil {
				ui.StopSpinnerFail(s, "Failed to merge PR")
				return fmt.Errorf("failed to merge: %w", err)
			}
			ui.StopSpinner(s, fmt.Sprintf("Merged PR #%d via %s", pr.Number, mergeMethod))
		}
	}

	// Switch to base branch
	s := ui.StartSpinner(fmt.Sprintf("Switching to %s...", baseBranch))
	if err := g.Checkout(baseBranch); err != nil {
		ui.StopSpinnerFail(s, fmt.Sprintf("Failed to switch to %s", baseBranch))
		return err
	}
	ui.StopSpinner(s, fmt.Sprintf("Switched to %s", baseBranch))

	// Pull latest
	s = ui.StartSpinner("Pulling latest changes...")
	if err := g.PullRebase(); err != nil {
		ui.StopSpinnerFail(s, "Failed to pull")
		ui.Warn("You may need to pull manually")
	} else {
		ui.StopSpinner(s, "Pulled latest changes")
	}

	// Delete local branch
	if !finishNoDelete {
		s = ui.StartSpinner(fmt.Sprintf("Deleting local branch %s...", branch))
		if err := g.DeleteBranch(branch); err != nil {
			// Try force delete
			if err := g.ForceDeleteBranch(branch); err != nil {
				ui.StopSpinnerFail(s, "Failed to delete local branch")
			} else {
				ui.StopSpinner(s, fmt.Sprintf("Deleted local branch %s", branch))
			}
		} else {
			ui.StopSpinner(s, fmt.Sprintf("Deleted local branch %s", branch))
		}
	}

	// Run post-merge hooks
	if len(cfg.Hooks.PostMerge) > 0 {
		fmt.Println()
		ui.Info("Running post-merge hooks...")
		for _, hook := range cfg.Hooks.PostMerge {
			if err := runHook(hook); err != nil {
				ui.Warn(fmt.Sprintf("Hook failed: %s", hook))
			}
		}
	}

	fmt.Println()
	ui.SuccessMsg(fmt.Sprintf("Finished %s", branch))
	ui.Detail("Merged into", baseBranch)
	ui.Detail("Method", mergeMethod)
	if !finishNoDelete {
		ui.Detail("Branch", "deleted")
	}
	fmt.Println()

	return nil
}
