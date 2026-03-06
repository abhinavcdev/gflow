package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/abhinavcdev/gflow/internal/git"
	"github.com/abhinavcdev/gflow/internal/ui"
	"github.com/spf13/cobra"
)

var checkoutCmd = &cobra.Command{
	Use:     "checkout [query]",
	Aliases: []string{"co", "switch"},
	Short:   "Fuzzy-switch between branches with an interactive picker",
	Long: `Quickly switch branches using fuzzy search.
Shows all local branches sorted by most recent, with type indicators.

Without arguments, opens an interactive picker.
With arguments, filters branches matching the query.

Examples:
  gflow checkout                  # Interactive branch picker
  gflow checkout auth             # Switch to branch matching "auth"
  gflow checkout feature/user     # Direct checkout
  gflow co auth                   # Short alias`,
	RunE: runCheckout,
}

var (
	checkoutRemote bool
)

func init() {
	checkoutCmd.Flags().BoolVarP(&checkoutRemote, "remote", "r", false, "include remote branches")
}

func runCheckout(cmd *cobra.Command, args []string) error {
	g := git.NewFromCwd()

	if !g.IsRepo() {
		ui.Error("Not a git repository")
		return fmt.Errorf("not a git repository")
	}

	// Get current branch
	currentBranch, _ := g.CurrentBranch()

	// Get branches
	branches, err := listBranches(g, checkoutRemote)
	if err != nil {
		ui.Error("Failed to list branches")
		return err
	}

	if len(branches) == 0 {
		ui.Info("No branches found")
		return nil
	}

	// If a query is provided, filter and auto-select
	if len(args) > 0 {
		query := strings.Join(args, " ")

		// Exact match first
		for _, b := range branches {
			if b == query {
				return doCheckout(g, b, currentBranch)
			}
		}

		// Fuzzy filter
		filtered := fuzzyFilter(branches, query)
		if len(filtered) == 0 {
			ui.Errorf("No branches matching '%s'", query)
			return fmt.Errorf("no match for '%s'", query)
		}
		if len(filtered) == 1 {
			return doCheckout(g, filtered[0], currentBranch)
		}

		// Multiple matches — interactive select
		branches = filtered
	}

	// Format branches for display
	items := make([]string, 0, len(branches))
	for _, b := range branches {
		label := formatBranchLabel(b, currentBranch)
		items = append(items, label)
	}

	idx, _, err := ui.PromptSelect("Switch to branch", items)
	if err != nil {
		return nil
	}

	return doCheckout(g, branches[idx], currentBranch)
}

func doCheckout(g *git.Git, branch, currentBranch string) error {
	if branch == currentBranch {
		ui.Info(fmt.Sprintf("Already on %s", branch))
		return nil
	}

	// Check for uncommitted changes
	if g.HasChanges() {
		ui.Warn("Uncommitted changes detected")
		stash, err := ui.PromptConfirm("Stash changes before switching?", true)
		if err != nil {
			return err
		}
		if stash {
			if err := g.Stash(fmt.Sprintf("gflow checkout: auto-stash from %s", currentBranch)); err != nil {
				ui.StepFail("Failed to stash")
				return err
			}
			ui.StepDone("Stashed changes")
		}
	}

	s := ui.StartSpinner(fmt.Sprintf("Switching to %s...", branch))

	// Strip remote prefix if present
	cleanBranch := branch
	if strings.HasPrefix(branch, "origin/") {
		cleanBranch = strings.TrimPrefix(branch, "origin/")
	}

	if err := g.Checkout(cleanBranch); err != nil {
		ui.StopSpinnerFail(s, fmt.Sprintf("Failed to switch to %s", cleanBranch))
		return err
	}
	ui.StopSpinner(s, fmt.Sprintf("Switched to %s", cleanBranch))

	// Show branch info
	branchType, name := parseBranchName(cleanBranch)
	if name != cleanBranch {
		ui.Detail("Type", branchType)
		ui.Detail("Name", name)
	}

	fmt.Println()
	return nil
}

func listBranches(g *git.Git, includeRemote bool) ([]string, error) {
	args := "branch"
	if includeRemote {
		args = "branch -a"
	}

	out, err := g.Run(strings.Fields(args)...)
	if err != nil {
		return nil, err
	}

	var branches []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip current branch marker
		line = strings.TrimPrefix(line, "* ")
		// Skip HEAD pointer
		if strings.Contains(line, "->") {
			continue
		}
		// Clean remote prefix for display
		line = strings.TrimPrefix(line, "remotes/")
		branches = append(branches, line)
	}

	// Sort: feature branches first, then others
	sort.Slice(branches, func(i, j int) bool {
		iPriority := branchSortPriority(branches[i])
		jPriority := branchSortPriority(branches[j])
		if iPriority != jPriority {
			return iPriority < jPriority
		}
		return branches[i] < branches[j]
	})

	return branches, nil
}

func branchSortPriority(branch string) int {
	switch {
	case strings.HasPrefix(branch, "feature/") || strings.HasPrefix(branch, "feat/"):
		return 1
	case strings.HasPrefix(branch, "bugfix/") || strings.HasPrefix(branch, "fix/"):
		return 2
	case strings.HasPrefix(branch, "hotfix/") || strings.HasPrefix(branch, "hot/"):
		return 3
	case strings.HasPrefix(branch, "release/"):
		return 4
	case branch == "main" || branch == "master":
		return 0
	case branch == "develop" || branch == "dev":
		return 0
	case strings.HasPrefix(branch, "origin/"):
		return 10
	default:
		return 5
	}
}

func fuzzyFilter(branches []string, query string) []string {
	query = strings.ToLower(query)
	var matches []string

	for _, b := range branches {
		lower := strings.ToLower(b)
		if strings.Contains(lower, query) {
			matches = append(matches, b)
		}
	}

	// If no substring matches, try fuzzy
	if len(matches) == 0 {
		for _, b := range branches {
			if fuzzyMatch(strings.ToLower(b), query) {
				matches = append(matches, b)
			}
		}
	}

	return matches
}

func fuzzyMatch(str, pattern string) bool {
	pi := 0
	for si := 0; si < len(str) && pi < len(pattern); si++ {
		if str[si] == pattern[pi] {
			pi++
		}
	}
	return pi == len(pattern)
}

func formatBranchLabel(branch, current string) string {
	label := branch
	if branch == current {
		label = fmt.Sprintf("%s (current)", branch)
	}

	branchType, _ := parseBranchName(branch)
	switch branchType {
	case "feature", "feat":
		return fmt.Sprintf("✨ %s", label)
	case "bugfix", "fix":
		return fmt.Sprintf("🐛 %s", label)
	case "hotfix", "hot":
		return fmt.Sprintf("🔥 %s", label)
	case "release":
		return fmt.Sprintf("🏷  %s", label)
	default:
		if branch == "main" || branch == "master" || branch == "develop" {
			return fmt.Sprintf("🔒 %s", label)
		}
		return fmt.Sprintf("   %s", label)
	}
}
