package cmd

import (
	"fmt"
	"strings"

	"github.com/abhinavcdev/gflow/internal/git"
	"github.com/abhinavcdev/gflow/internal/ui"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff [base]",
	Short: "Show a summary of changes between your branch and base",
	Long: `Displays a compact diff summary showing files changed, insertions, 
and deletions compared to the base branch. Useful before opening a PR.

Examples:
  gflow diff              # Diff against default base
  gflow diff main         # Diff against specific branch
  gflow diff --stat       # Show only file stats
  gflow diff --files      # Show only changed file names`,
	RunE: runDiff,
}

var (
	diffStat  bool
	diffFiles bool
	diffFull  bool
)

func init() {
	diffCmd.Flags().BoolVar(&diffStat, "stat", false, "show only diff statistics")
	diffCmd.Flags().BoolVar(&diffFiles, "files", false, "show only changed file names")
	diffCmd.Flags().BoolVar(&diffFull, "full", false, "show full diff output")
}

func runDiff(cmd *cobra.Command, args []string) error {
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

	// Determine base
	base := ""
	if len(args) > 0 {
		base = args[0]
	} else {
		base = detectBaseBranch(currentBranch)
	}

	ui.Title(fmt.Sprintf("  Changes: %s → %s", currentBranch, base))
	fmt.Println()

	// Commit count
	commitCount, err := g.CommitCount(base)
	if err == nil {
		ui.Detail("Commits ahead", fmt.Sprintf("%d", commitCount))
	}

	if diffFiles {
		// Show only file names
		out, err := g.Run("diff", "--name-only", base+"...")
		if err != nil {
			ui.Error("Failed to get diff")
			return err
		}
		if out == "" {
			ui.Info("No changes")
			return nil
		}

		files := strings.Split(out, "\n")
		ui.Detail("Files changed", fmt.Sprintf("%d", len(files)))
		fmt.Println()

		for _, f := range files {
			f = strings.TrimSpace(f)
			if f == "" {
				continue
			}
			var icon string
			if strings.HasSuffix(f, "_test.go") || strings.Contains(f, "test") {
				icon = "🧪"
			} else if strings.HasSuffix(f, ".md") {
				icon = "📖"
			} else if strings.HasSuffix(f, ".yml") || strings.HasSuffix(f, ".yaml") || strings.HasSuffix(f, ".json") {
				icon = "⚙️ "
			} else {
				icon = "📄"
			}
			fmt.Printf("  %s %s\n", icon, f)
		}
		fmt.Println()
		return nil
	}

	if diffFull {
		// Full diff with color
		out, err := g.Run("diff", "--color=always", base+"...")
		if err != nil {
			ui.Error("Failed to get diff")
			return err
		}
		fmt.Println(out)
		return nil
	}

	// Default: stat view
	stat, err := g.DiffStat(base)
	if err != nil {
		ui.Error("Failed to get diff stat")
		return err
	}

	if stat == "" {
		ui.Info("No changes between branches")
		fmt.Println()
		return nil
	}

	// Parse and colorize stat output
	lines := strings.Split(stat, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if i == len(lines)-1 {
			// Summary line
			fmt.Println()
			fmt.Printf("  %s\n", ui.BoldStyle.Render(line))
		} else {
			// File line — colorize insertions/deletions
			parts := strings.SplitN(line, "|", 2)
			if len(parts) == 2 {
				fileName := strings.TrimSpace(parts[0])
				changes := strings.TrimSpace(parts[1])

				colorized := colorizeDiffBar(changes)
				fmt.Printf("  %s %s %s\n",
					ui.MutedStyle.Render("|"),
					fmt.Sprintf("%-50s", fileName),
					colorized,
				)
			} else {
				fmt.Printf("  %s\n", line)
			}
		}
	}

	fmt.Println()
	return nil
}

func colorizeDiffBar(changes string) string {
	result := ""
	for _, ch := range changes {
		switch ch {
		case '+':
			result += ui.SuccessStyle.Render("+")
		case '-':
			result += ui.ErrorStyle.Render("-")
		default:
			result += string(ch)
		}
	}
	return result
}
