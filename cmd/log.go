package cmd

import (
	"fmt"
	"strings"

	"github.com/abhinavcdev/gflow/internal/git"
	"github.com/abhinavcdev/gflow/internal/ui"
	"github.com/spf13/cobra"
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Pretty commit log with branch context and PR links",
	Long: `Show a formatted commit log for the current branch, highlighting
commits since branching from the base. Includes commit types, 
branch info, and links to associated PRs.

Examples:
  gflow log              # Show commits since base branch
  gflow log -n 20        # Show last 20 commits
  gflow log --all        # Show full log, not just since base`,
	RunE: runLog,
}

var (
	logCount int
	logAll   bool
)

func init() {
	logCmd.Flags().IntVarP(&logCount, "count", "n", 15, "number of commits to show")
	logCmd.Flags().BoolVar(&logAll, "all", false, "show all commits, not just since base")
}

func runLog(cmd *cobra.Command, args []string) error {
	g := git.NewFromCwd()

	if !g.IsRepo() {
		ui.Error("Not a git repository")
		return fmt.Errorf("not a git repository")
	}

	currentBranch, err := g.CurrentBranch()
	if err != nil {
		ui.Error("Failed to determine current branch")
		return err
	}

	baseBranch := detectBaseBranch(currentBranch)

	ui.Title(fmt.Sprintf("  Commit Log — %s", currentBranch))
	fmt.Println()

	var commits string
	if logAll || currentBranch == cfg.Branching.Main || currentBranch == cfg.Branching.Develop {
		commits, err = g.Log(logCount, "%h||%s||%an||%ar")
	} else {
		commits, err = g.LogBetween(baseBranch, "HEAD", "%h||%s||%an||%ar")
		if err != nil || commits == "" {
			// Fallback to regular log
			commits, err = g.Log(logCount, "%h||%s||%an||%ar")
		}
	}

	if err != nil {
		ui.Error("Failed to read commit log")
		return err
	}

	if commits == "" {
		ui.Info("No commits found")
		return nil
	}

	for _, line := range strings.Split(commits, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "||", 4)
		if len(parts) < 4 {
			fmt.Printf("  %s\n", line)
			continue
		}

		hash := parts[0]
		subject := parts[1]
		author := parts[2]
		relTime := parts[3]

		// Colorize by commit type
		icon := " "
		commitType := extractCommitType(subject)
		switch commitType {
		case "feat":
			icon = ui.SuccessStyle.Render("✨")
		case "fix":
			icon = ui.ErrorStyle.Render("🐛")
		case "docs":
			icon = ui.MutedStyle.Render("📖")
		case "refactor":
			icon = ui.SubtitleStyle.Render("♻️ ")
		case "test":
			icon = ui.WarningStyle.Render("🧪")
		case "ci", "build":
			icon = ui.MutedStyle.Render("⚙️ ")
		case "chore":
			icon = ui.MutedStyle.Render("🔧")
		case "perf":
			icon = ui.WarningStyle.Render("⚡")
		case "style":
			icon = ui.MutedStyle.Render("💄")
		case "revert":
			icon = ui.ErrorStyle.Render("⏪")
		}

		hashStyled := ui.WarningStyle.Render(hash)
		authorStyled := ui.MutedStyle.Render(author)
		timeStyled := ui.MutedStyle.Render(relTime)

		fmt.Printf("  %s %s %s  %s  %s\n", icon, hashStyled, subject, authorStyled, timeStyled)
	}

	fmt.Println()

	// Show summary
	if !logAll && currentBranch != cfg.Branching.Main && currentBranch != cfg.Branching.Develop {
		count, err := g.CommitCount(baseBranch)
		if err == nil {
			ui.Detail("Commits ahead of "+baseBranch, fmt.Sprintf("%d", count))
		}
	}

	diffStat, err := g.DiffStat(baseBranch)
	if err == nil && diffStat != "" {
		lines := strings.Split(strings.TrimSpace(diffStat), "\n")
		if len(lines) > 0 {
			// Show the summary line (last line of diffstat)
			ui.Detail("Changes", strings.TrimSpace(lines[len(lines)-1]))
		}
	}

	fmt.Println()
	return nil
}

func extractCommitType(subject string) string {
	// Match conventional commits: type(scope): message or type: message
	subject = strings.TrimSpace(subject)
	for _, sep := range []string{"(", ":"} {
		if idx := strings.Index(subject, sep); idx > 0 && idx < 15 {
			candidate := strings.ToLower(subject[:idx])
			switch candidate {
			case "feat", "fix", "docs", "style", "refactor", "perf", "test", "build", "ci", "chore", "revert":
				return candidate
			}
		}
	}
	return ""
}
