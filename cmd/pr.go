package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/abhinavcdev/gflow/internal/git"
	"github.com/abhinavcdev/gflow/internal/provider"
	"github.com/abhinavcdev/gflow/internal/ui"
	"github.com/spf13/cobra"
)

var prCmd = &cobra.Command{
	Use:   "pr [title]",
	Short: "Create a pull request from the current branch",
	Long: `Creates a pull request with your team's conventions — all in one command.

Stages changes, commits, pushes, opens the PR with template, assigns reviewers, 
adds labels, and prints the PR URL.

Examples:
  gflow pr                              # Interactive PR creation
  gflow pr "Add user authentication"    # With title
  gflow pr --draft                      # Create as draft
  gflow pr --base main                  # Target specific base branch
  gflow pr -r alice -r bob              # Assign reviewers
  gflow pr --label bug --label urgent   # Add labels`,
	RunE: runPR,
}

var (
	prBase          string
	prDraft         bool
	prReviewers     []string
	prTeamReviewers []string
	prLabels        []string
	prBody          string
	prNoEdit        bool
	prCommitMsg     string
	prPush          bool
)

func init() {
	prCmd.Flags().StringVarP(&prBase, "base", "b", "", "base branch for the PR")
	prCmd.Flags().BoolVarP(&prDraft, "draft", "d", false, "create as draft PR")
	prCmd.Flags().StringArrayVarP(&prReviewers, "reviewer", "r", nil, "add reviewers")
	prCmd.Flags().StringArrayVarP(&prTeamReviewers, "team-reviewer", "t", nil, "add team reviewers")
	prCmd.Flags().StringArrayVarP(&prLabels, "label", "l", nil, "add labels")
	prCmd.Flags().StringVar(&prBody, "body", "", "PR body/description")
	prCmd.Flags().BoolVar(&prNoEdit, "no-edit", false, "skip interactive editing")
	prCmd.Flags().StringVarP(&prCommitMsg, "message", "m", "", "commit message (stages and commits before PR)")
	prCmd.Flags().BoolVar(&prPush, "push", true, "push before creating PR")

	prCmd.AddCommand(prListCmd)
	prCmd.AddCommand(prViewCmd)
}

var prListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List open pull requests",
	Long: `Show all open pull requests for the current repository in a formatted table.

Examples:
  gflow pr list
  gflow pr ls`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.Title("  Open Pull Requests")
		fmt.Println()

		p, err := provider.New(cfg)
		if err != nil {
			ui.Error("Failed to connect to provider")
			return err
		}

		s := ui.StartSpinner("Fetching pull requests...")
		prs, err := p.ListPRs()
		if err != nil {
			ui.StopSpinnerFail(s, "Failed to fetch PRs")
			return err
		}
		ui.StopSpinner(s, fmt.Sprintf("Found %d open PR(s)", len(prs)))
		fmt.Println()

		if len(prs) == 0 {
			ui.Info("No open pull requests")
			fmt.Println()
			return nil
		}

		// Detect current branch for highlighting
		g := git.NewFromCwd()
		currentBranch, _ := g.CurrentBranch()

		for _, pr := range prs {
			icon := "  "
			if pr.Draft {
				icon = ui.MutedStyle.Render("◌")
			} else {
				icon = ui.SuccessStyle.Render("●")
			}

			number := ui.BoldStyle.Render(fmt.Sprintf("#%-5d", pr.Number))
			title := pr.Title
			if len(title) > 60 {
				title = title[:57] + "..."
			}

			branchInfo := ui.MutedStyle.Render(fmt.Sprintf("%s → %s", pr.Head, pr.Base))

			marker := ""
			if pr.Head == currentBranch {
				marker = ui.SuccessStyle.Render(" ← you")
			}

			labelStr := ""
			if len(pr.Labels) > 0 {
				labelStr = " " + ui.MutedStyle.Render("["+strings.Join(pr.Labels, ", ")+"]")
			}

			fmt.Printf("  %s %s %s%s%s\n", icon, number, title, marker, labelStr)
			fmt.Printf("           %s\n", branchInfo)
		}

		fmt.Println()
		return nil
	},
}

var prViewCmd = &cobra.Command{
	Use:   "view [number]",
	Short: "View detailed info about a pull request",
	Long: `Show detailed information about a PR. Without arguments, shows 
the PR for the current branch.

Examples:
  gflow pr view        # View PR for current branch
  gflow pr view 42     # View PR #42`,
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := provider.New(cfg)
		if err != nil {
			ui.Error("Failed to connect to provider")
			return err
		}

		var pr *provider.PullRequest

		if len(args) > 0 {
			var num int
			if _, err := fmt.Sscanf(args[0], "%d", &num); err != nil {
				ui.Errorf("Invalid PR number: %s", args[0])
				return err
			}
			pr, err = p.GetPR(num)
			if err != nil {
				ui.Errorf("Failed to fetch PR #%d", num)
				return err
			}
		} else {
			g := git.NewFromCwd()
			branch, err := g.CurrentBranch()
			if err != nil {
				ui.Error("Failed to determine current branch")
				return err
			}
			pr, err = p.GetPRForBranch(branch)
			if err != nil {
				ui.Errorf("No open PR found for branch %s", branch)
				return err
			}
		}

		ui.Title(fmt.Sprintf("  PR #%d", pr.Number))
		fmt.Println()

		stateIcon := ui.SuccessStyle.Render("●")
		stateLabel := "Open"
		if pr.Draft {
			stateIcon = ui.MutedStyle.Render("◌")
			stateLabel = "Draft"
		}
		if pr.State == "closed" || pr.State == "DECLINED" {
			stateIcon = ui.ErrorStyle.Render("●")
			stateLabel = "Closed"
		}
		if pr.State == "merged" || pr.State == "MERGED" {
			stateIcon = ui.SubtitleStyle.Render("●")
			stateLabel = "Merged"
		}

		fmt.Printf("  %s %s  %s\n", stateIcon, stateLabel, ui.BoldStyle.Render(pr.Title))
		fmt.Println()

		ui.Detail("Number", fmt.Sprintf("#%d", pr.Number))
		ui.Detail("Branch", fmt.Sprintf("%s → %s", pr.Head, pr.Base))
		ui.Detail("URL", pr.URL)

		if len(pr.Reviewers) > 0 {
			ui.Detail("Reviewers", strings.Join(pr.Reviewers, ", "))
		}
		if len(pr.Labels) > 0 {
			ui.Detail("Labels", strings.Join(pr.Labels, ", "))
		}

		if pr.Body != "" {
			fmt.Println()
			ui.Divider()
			fmt.Println()
			// Truncate very long bodies
			body := pr.Body
			if len(body) > 1000 {
				body = body[:997] + "..."
			}
			fmt.Println(body)
		}

		fmt.Println()
		return nil
	},
}

func runPR(cmd *cobra.Command, args []string) error {
	g := git.NewFromCwd()

	if !g.IsRepo() {
		ui.Error("Not a git repository")
		return fmt.Errorf("not a git repository")
	}

	ui.Title("  Create Pull Request")
	fmt.Println()

	// Get current branch
	currentBranch, err := g.CurrentBranch()
	if err != nil {
		ui.Error("Failed to determine current branch")
		return err
	}

	// Don't allow PR from main/develop
	if currentBranch == cfg.Branching.Main || currentBranch == cfg.Branching.Develop {
		ui.Errorf("Cannot create PR from %s — switch to a feature branch first", currentBranch)
		return fmt.Errorf("cannot create PR from %s", currentBranch)
	}

	// Determine base branch
	baseBranch := cfg.PR.DefaultBase
	if prBase != "" {
		baseBranch = prBase
	} else {
		// Auto-detect base from branch type
		baseBranch = detectBaseBranch(currentBranch)
	}

	// Step 1: Handle uncommitted changes
	if g.HasChanges() {
		ui.Step(1, "Uncommitted changes detected")

		if prCommitMsg == "" {
			// Stage all changes
			status, _ := g.Status()
			fmt.Println(ui.MutedStyle.Render("    " + strings.ReplaceAll(status, "\n", "\n    ")))
			fmt.Println()

			stageAll, err := ui.PromptConfirm("Stage all changes?", true)
			if err != nil {
				return err
			}

			if stageAll {
				if err := g.AddAll(); err != nil {
					ui.StepFail("Failed to stage changes")
					return err
				}
			}

			// Get commit message
			defaultMsg := generateCommitMessage(currentBranch)
			prCommitMsg, err = ui.PromptInputRequired("Commit message", defaultMsg)
			if err != nil {
				return err
			}
		} else {
			// Auto-stage and commit
			if err := g.AddAll(); err != nil {
				ui.StepFail("Failed to stage changes")
				return err
			}
		}

		// Commit
		s := ui.StartSpinner("Committing changes...")
		if err := g.Commit(prCommitMsg); err != nil {
			ui.StopSpinnerFail(s, "Failed to commit")
			return err
		}
		ui.StopSpinner(s, "Committed changes")
	} else if g.HasStagedChanges() {
		ui.Step(1, "Staged changes detected")

		if prCommitMsg == "" {
			defaultMsg := generateCommitMessage(currentBranch)
			var err error
			prCommitMsg, err = ui.PromptInputRequired("Commit message", defaultMsg)
			if err != nil {
				return err
			}
		}

		s := ui.StartSpinner("Committing staged changes...")
		if err := g.Commit(prCommitMsg); err != nil {
			ui.StopSpinnerFail(s, "Failed to commit")
			return err
		}
		ui.StopSpinner(s, "Committed staged changes")
	} else {
		ui.StepDone("Working tree clean")
	}

	// Step 2: Push to remote
	if prPush {
		s := ui.StartSpinner("Pushing to remote...")
		err := g.PushSetUpstream()
		if err != nil {
			// Try regular push
			err = g.Push()
		}
		if err != nil {
			ui.StopSpinnerFail(s, "Failed to push")
			return fmt.Errorf("failed to push: %w", err)
		}
		ui.StopSpinner(s, fmt.Sprintf("Pushed %s to origin", currentBranch))
	}

	// Step 3: Build PR title
	var prTitle string
	if len(args) > 0 {
		prTitle = strings.Join(args, " ")
	}

	if prTitle == "" && !prNoEdit {
		defaultTitle := generatePRTitle(currentBranch)
		prTitle, err = ui.PromptInputRequired("PR title", defaultTitle)
		if err != nil {
			return err
		}
	} else if prTitle == "" {
		prTitle = generatePRTitle(currentBranch)
	}

	// Step 4: Build PR body
	body := prBody
	if body == "" {
		body = generatePRBody(currentBranch, baseBranch, g)
	}

	if !prNoEdit && prBody == "" {
		editBody, err := ui.PromptConfirm("Edit PR description?", false)
		if err != nil {
			return err
		}
		if editBody {
			edited, err := ui.PromptInput("PR description (single line, or leave empty for template)", "")
			if err != nil {
				return err
			}
			if edited != "" {
				body = edited
			}
		}
	}

	// Step 5: Determine reviewers and labels
	reviewers := prReviewers
	if len(reviewers) == 0 {
		reviewers = cfg.PR.Reviewers
	}

	teamReviewers := prTeamReviewers
	if len(teamReviewers) == 0 {
		teamReviewers = cfg.PR.TeamReviewers
	}

	labels := prLabels
	if len(labels) == 0 {
		labels = cfg.PR.Labels
		// Auto-add labels based on branch type
		autoLabels := detectLabels(currentBranch)
		labels = append(labels, autoLabels...)
	}

	isDraft := prDraft || cfg.PR.Draft

	// Show summary before creating
	fmt.Println()
	ui.Divider()
	ui.OrderedSummary("PR Summary", []ui.SummaryItem{
		{Label: "Title", Value: prTitle},
		{Label: "Branch", Value: fmt.Sprintf("%s → %s", currentBranch, baseBranch)},
		{Label: "Draft", Value: fmt.Sprintf("%v", isDraft)},
		{Label: "Reviewers", Value: formatList(reviewers)},
		{Label: "Labels", Value: formatList(labels)},
	})

	// Step 6: Create the PR
	s := ui.StartSpinner("Creating pull request...")

	p, err := provider.New(cfg)
	if err != nil {
		ui.StopSpinnerFail(s, "Failed to connect to provider")
		return err
	}

	pr, err := p.CreatePR(provider.PRCreateOptions{
		Title:         prTitle,
		Body:          body,
		Head:          currentBranch,
		Base:          baseBranch,
		Draft:         isDraft,
		Labels:        labels,
		Reviewers:     reviewers,
		TeamReviewers: teamReviewers,
		MergeMethod:   cfg.PR.MergeMethod,
	})
	if err != nil {
		ui.StopSpinnerFail(s, "Failed to create PR")
		return fmt.Errorf("failed to create PR: %w", err)
	}

	ui.StopSpinner(s, "Pull request created!")

	// Success output
	fmt.Println()
	ui.SuccessMsg(fmt.Sprintf("PR #%d created", pr.Number))
	ui.Detail("URL", pr.URL)
	ui.Detail("Title", pr.Title)
	ui.Detail("Branch", fmt.Sprintf("%s → %s", pr.Head, pr.Base))
	if pr.Draft {
		ui.Detail("Status", "Draft")
	}
	fmt.Println()

	// Run post-PR hooks
	if len(cfg.Hooks.PostPR) > 0 {
		for _, hook := range cfg.Hooks.PostPR {
			_ = runHook(hook)
		}
	}

	return nil
}

func detectBaseBranch(branch string) string {
	if cfg == nil {
		return "main"
	}

	// Hotfix branches always target main
	if strings.HasPrefix(branch, cfg.Branching.Prefixes.Hotfix) {
		return cfg.Branching.Main
	}

	// Release branches target main
	if strings.HasPrefix(branch, cfg.Branching.Prefixes.Release) {
		return cfg.Branching.Main
	}

	return cfg.GetBaseBranch()
}

func generateCommitMessage(branch string) string {
	if cfg == nil || cfg.Commit.Convention == "none" {
		return ""
	}

	// Extract type and name from branch
	branchType, name := parseBranchName(branch)

	switch cfg.Commit.Convention {
	case "conventional":
		commitType := mapBranchToCommitType(branchType)
		return fmt.Sprintf("%s: %s", commitType, humanize(name))
	case "angular":
		commitType := mapBranchToCommitType(branchType)
		return fmt.Sprintf("%s: %s", commitType, humanize(name))
	default:
		return humanize(name)
	}
}

func generatePRTitle(branch string) string {
	branchType, name := parseBranchName(branch)

	if cfg != nil && cfg.PR.TitleFormat != "" {
		tmpl, err := template.New("title").Parse(cfg.PR.TitleFormat)
		if err == nil {
			data := map[string]string{
				"Type":        mapBranchToCommitType(branchType),
				"Description": humanize(name),
				"Branch":      branch,
			}
			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, data); err == nil {
				return buf.String()
			}
		}
	}

	commitType := mapBranchToCommitType(branchType)
	return fmt.Sprintf("%s: %s", commitType, humanize(name))
}

func generatePRBody(branch, base string, g *git.Git) string {
	// Check for template file
	if cfg != nil && cfg.PR.TemplatePath != "" {
		if data, err := os.ReadFile(cfg.PR.TemplatePath); err == nil {
			return string(data)
		}
	}

	// Check common PR template locations
	templatePaths := []string{
		".github/pull_request_template.md",
		".github/PULL_REQUEST_TEMPLATE.md",
		"docs/pull_request_template.md",
		".gitlab/merge_request_templates/Default.md",
	}
	for _, p := range templatePaths {
		if data, err := os.ReadFile(p); err == nil {
			return string(data)
		}
	}

	// Generate a default body
	_, name := parseBranchName(branch)

	var body strings.Builder
	body.WriteString(fmt.Sprintf("## %s\n\n", humanize(name)))
	body.WriteString("### Changes\n\n")

	// Add commit log
	commits, err := g.LogBetween(base, "HEAD", "%s")
	if err == nil && commits != "" {
		for _, line := range strings.Split(commits, "\n") {
			if line != "" {
				body.WriteString(fmt.Sprintf("- %s\n", line))
			}
		}
	} else {
		body.WriteString("- \n")
	}

	body.WriteString("\n### Testing\n\n")
	body.WriteString("- [ ] Unit tests\n")
	body.WriteString("- [ ] Integration tests\n")
	body.WriteString("- [ ] Manual testing\n")
	body.WriteString("\n### Notes\n\n")
	body.WriteString("_No additional notes._\n")

	return body.String()
}

func parseBranchName(branch string) (branchType, name string) {
	if cfg != nil {
		prefixes := map[string]string{
			"feature": cfg.Branching.Prefixes.Feature,
			"bugfix":  cfg.Branching.Prefixes.Bugfix,
			"hotfix":  cfg.Branching.Prefixes.Hotfix,
			"release": cfg.Branching.Prefixes.Release,
			"support": cfg.Branching.Prefixes.Support,
		}
		for t, prefix := range prefixes {
			if prefix != "" && strings.HasPrefix(branch, prefix) {
				return t, strings.TrimPrefix(branch, prefix)
			}
		}
	}

	// Try common prefixes
	for _, prefix := range []string{"feature/", "feat/", "bugfix/", "fix/", "hotfix/", "hot/", "release/", "support/"} {
		if strings.HasPrefix(branch, prefix) {
			parts := strings.SplitN(branch, "/", 2)
			return parts[0], parts[1]
		}
	}

	return "feature", branch
}

func mapBranchToCommitType(branchType string) string {
	switch branchType {
	case "feature", "feat":
		return "feat"
	case "bugfix", "fix":
		return "fix"
	case "hotfix", "hot":
		return "fix"
	case "release":
		return "release"
	case "support":
		return "chore"
	default:
		return "feat"
	}
}

func humanize(slug string) string {
	s := strings.ReplaceAll(slug, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.TrimSpace(s)
	if len(s) > 0 {
		return strings.ToUpper(s[:1]) + s[1:]
	}
	return s
}

func detectLabels(branch string) []string {
	branchType, _ := parseBranchName(branch)
	switch branchType {
	case "bugfix", "fix":
		return []string{"bug"}
	case "hotfix", "hot":
		return []string{"hotfix", "priority"}
	case "feature", "feat":
		return []string{"enhancement"}
	default:
		return nil
	}
}

func formatList(items []string) string {
	if len(items) == 0 {
		return "(none)"
	}
	return strings.Join(items, ", ")
}
