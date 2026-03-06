package cmd

import (
	"fmt"
	"strings"

	"github.com/abhinavcdev/gflow/internal/git"
	"github.com/abhinavcdev/gflow/internal/provider"
	"github.com/abhinavcdev/gflow/internal/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the current branch status, PR info, and sync state",
	Long: `Displays a comprehensive overview of your current working state:
- Current branch and type
- Uncommitted changes
- Commits ahead/behind base
- Associated pull request info
- Sync status with upstream

Examples:
  gflow status`,
	RunE: runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	g := git.NewFromCwd()

	if !g.IsRepo() {
		ui.Error("Not a git repository")
		return fmt.Errorf("not a git repository")
	}

	ui.Title("  Status")
	fmt.Println()

	// Current branch
	branch, err := g.CurrentBranch()
	if err != nil {
		ui.Error("Failed to determine current branch")
		return err
	}

	branchType, name := parseBranchName(branch)
	ui.Detail("Branch", branch)
	ui.Detail("Type", branchType)
	if name != branch {
		ui.Detail("Name", name)
	}

	// Base branch
	baseBranch := detectBaseBranch(branch)
	ui.Detail("Base", baseBranch)

	fmt.Println()

	// Working tree status
	if g.HasChanges() {
		status, _ := g.Status()
		lines := strings.Split(status, "\n")
		staged := 0
		modified := 0
		untracked := 0
		for _, l := range lines {
			l = strings.TrimSpace(l)
			if l == "" {
				continue
			}
			switch {
			case strings.HasPrefix(l, "A ") || strings.HasPrefix(l, "M ") || strings.HasPrefix(l, "D "):
				staged++
			case strings.HasPrefix(l, " M") || strings.HasPrefix(l, " D"):
				modified++
			case strings.HasPrefix(l, "??"):
				untracked++
			default:
				modified++
			}
		}
		ui.Warn(fmt.Sprintf("Working tree: %d staged, %d modified, %d untracked", staged, modified, untracked))
	} else {
		ui.StepDone("Working tree clean")
	}

	// Commits ahead of base
	commitCount, err := g.CommitCount(baseBranch)
	if err == nil {
		if commitCount > 0 {
			ui.Info(fmt.Sprintf("%d commit(s) ahead of %s", commitCount, baseBranch))
		} else {
			ui.StepDone(fmt.Sprintf("Up to date with %s", baseBranch))
		}
	}

	// Last commit
	lastMsg, err := g.LastCommitMessage()
	if err == nil && lastMsg != "" {
		hash, _ := g.LastCommitHash()
		ui.Detail("Last commit", fmt.Sprintf("%s %s", ui.MutedStyle.Render(hash), lastMsg))
	}

	fmt.Println()

	// PR status
	if branch != cfg.Branching.Main && branch != cfg.Branching.Develop {
		p, err := provider.New(cfg)
		if err == nil {
			pr, err := p.GetPRForBranch(branch)
			if err == nil {
				ui.StepDone(fmt.Sprintf("PR #%d: %s", pr.Number, pr.Title))
				ui.Detail("URL", pr.URL)
				ui.Detail("State", pr.State)
				if pr.Draft {
					ui.Detail("Draft", "yes")
				}
				if len(pr.Reviewers) > 0 {
					ui.Detail("Reviewers", strings.Join(pr.Reviewers, ", "))
				}
				if len(pr.Labels) > 0 {
					ui.Detail("Labels", strings.Join(pr.Labels, ", "))
				}
			} else {
				ui.Info("No open PR for this branch")
				ui.Detail("Create one", "gflow pr")
			}
		} else {
			ui.MutedStyle.Render("  (provider not configured — skipping PR status)")
		}
	}

	fmt.Println()
	return nil
}
