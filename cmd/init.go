package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/abhinavcdev/gflow/internal/config"
	"github.com/abhinavcdev/gflow/internal/git"
	"github.com/abhinavcdev/gflow/internal/ui"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize gflow in the current repository",
	Long:  "Interactive setup wizard that creates a .gflow.yml configuration file with your team's conventions.",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	ui.Banner()
	ui.Title("Initialize gflow")
	fmt.Println()

	g := git.NewFromCwd()

	// Check if we're in a git repo
	if !g.IsRepo() {
		ui.Error("Not a git repository. Run 'git init' first.")
		return fmt.Errorf("not a git repository")
	}

	// Check if config already exists
	repoRoot, err := g.RepoRoot()
	if err != nil {
		repoRoot = "."
	}
	configPath := filepath.Join(repoRoot, config.ConfigFileName)

	if _, err := os.Stat(configPath); err == nil {
		overwrite, err := ui.PromptConfirm(fmt.Sprintf("%s already exists. Overwrite?", config.ConfigFileName), false)
		if err != nil || !overwrite {
			ui.Info("Aborted.")
			return nil
		}
	}

	cfg := config.DefaultConfig()

	// Step 1: Detect GitHub repo
	ui.Step(1, "GitHub Repository")
	remoteURL, err := g.RemoteURL()
	detectedOwner := ""
	detectedRepo := ""

	if err == nil {
		detectedOwner, detectedRepo, _ = git.ParseRemoteURL(remoteURL)
		ui.Detail("Detected", fmt.Sprintf("github.com/%s/%s", detectedOwner, detectedRepo))
	}

	cfg.Provider.Name = "github"
	cfg.Provider.TokenEnv = "GITHUB_TOKEN"

	// Owner and repo
	owner, err := ui.PromptInput("Repository owner/org", detectedOwner)
	if err != nil {
		return err
	}
	cfg.Provider.Owner = owner

	repo, err := ui.PromptInput("Repository name", detectedRepo)
	if err != nil {
		return err
	}
	cfg.Provider.Repo = repo

	// Check for GitHub token
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	if token != "" {
		ui.StepDone("GitHub token found")
	} else {
		ui.Warn("No GitHub token found")
		fmt.Println()
		fmt.Println(ui.BoldStyle.Render("  To generate a GitHub token:"))
		fmt.Println()
		fmt.Println("  1. Go to https://github.com/settings/tokens")
		fmt.Println("  2. Click \"Generate new token\" → \"Generate new token (classic)\"")
		fmt.Println("  3. Give it a name (e.g. \"gflow\")")
		fmt.Println("  4. Select scopes: " + ui.BoldStyle.Render("repo") + " (full control of private repos)")
		fmt.Println("  5. Click \"Generate token\" and copy it")
		fmt.Println()
		fmt.Println("  Then set it in your shell:")
		fmt.Println(ui.BoldStyle.Render("    export GITHUB_TOKEN=ghp_xxxxxxxxxxxx"))
		fmt.Println()
		fmt.Println(ui.MutedStyle.Render("  Add the export to ~/.zshrc or ~/.bashrc to make it permanent."))
		fmt.Println()
	}

	fmt.Println()

	// Step 2: Branching strategy
	ui.Step(2, "Branching Strategy")

	defaultBranch := g.DefaultBranch()
	mainBranch, err := ui.PromptInput("Main branch", defaultBranch)
	if err != nil {
		return err
	}
	cfg.Branching.Main = mainBranch
	cfg.PR.DefaultBase = mainBranch

	useDevelop, err := ui.PromptConfirm("Use a develop branch? (git-flow style)", false)
	if err != nil {
		return err
	}
	cfg.Branching.UseDevelop = useDevelop

	if useDevelop {
		developBranch, err := ui.PromptInput("Develop branch", "develop")
		if err != nil {
			return err
		}
		cfg.Branching.Develop = developBranch
		cfg.PR.DefaultBase = developBranch
	}

	_, prefixStyle, err := ui.PromptSelect("Branch prefix style", []string{
		"conventional (feature/, bugfix/, hotfix/)",
		"short (feat/, fix/, hot/)",
		"ticket-based (PROJ-123/)",
		"custom",
	})
	if err != nil {
		return err
	}

	switch prefixStyle {
	case "short (feat/, fix/, hot/)":
		cfg.Branching.Prefixes = config.BranchPrefixes{
			Feature: "feat/",
			Bugfix:  "fix/",
			Hotfix:  "hot/",
			Release: "release/",
			Support: "support/",
		}
	case "ticket-based (PROJ-123/)":
		cfg.Branching.Prefixes = config.BranchPrefixes{
			Feature: "",
			Bugfix:  "",
			Hotfix:  "hotfix/",
			Release: "release/",
			Support: "support/",
		}
	case "custom":
		feat, _ := ui.PromptInput("Feature prefix", "feature/")
		bug, _ := ui.PromptInput("Bugfix prefix", "bugfix/")
		hot, _ := ui.PromptInput("Hotfix prefix", "hotfix/")
		rel, _ := ui.PromptInput("Release prefix", "release/")
		cfg.Branching.Prefixes = config.BranchPrefixes{
			Feature: feat,
			Bugfix:  bug,
			Hotfix:  hot,
			Release: rel,
			Support: "support/",
		}
	}

	fmt.Println()

	// Step 3: PR conventions
	ui.Step(3, "Pull Request Conventions")

	_, mergeMethod, err := ui.PromptSelect("Default merge method", []string{"squash", "merge", "rebase"})
	if err != nil {
		return err
	}
	cfg.PR.MergeMethod = mergeMethod

	draft, err := ui.PromptConfirm("Create PRs as draft by default?", false)
	if err != nil {
		return err
	}
	cfg.PR.Draft = draft

	autoAssign, err := ui.PromptConfirm("Auto-assign PR creator?", true)
	if err != nil {
		return err
	}
	cfg.PR.AutoAssign = autoAssign

	deleteBranch, err := ui.PromptConfirm("Delete branch after merge?", true)
	if err != nil {
		return err
	}
	cfg.PR.DeleteBranch = deleteBranch

	reviewers, err := ui.PromptMultiInput("Default reviewers", "", "space-separated usernames, or empty")
	if err != nil {
		return err
	}
	cfg.PR.Reviewers = reviewers

	labels, err := ui.PromptMultiInput("Default labels", "", "space-separated labels, or empty")
	if err != nil {
		return err
	}
	cfg.PR.Labels = labels

	fmt.Println()

	// Step 4: Commit conventions
	ui.Step(4, "Commit Conventions")

	_, convention, err := ui.PromptSelect("Commit convention", []string{
		"conventional (feat: add feature)",
		"angular (feat(scope): add feature)",
		"none",
	})
	if err != nil {
		return err
	}

	switch convention {
	case "conventional (feat: add feature)":
		cfg.Commit.Convention = "conventional"
		cfg.Commit.RequireScope = false
	case "angular (feat(scope): add feature)":
		cfg.Commit.Convention = "angular"
		cfg.Commit.RequireScope = true
	case "none":
		cfg.Commit.Convention = "none"
	}

	requireTicket, err := ui.PromptConfirm("Require ticket number in branch name?", false)
	if err != nil {
		return err
	}
	cfg.Commit.RequireTicket = requireTicket

	if requireTicket {
		pattern, err := ui.PromptInput("Ticket pattern (regex)", `[A-Z]+-\d+`)
		if err != nil {
			return err
		}
		cfg.Commit.TicketPattern = pattern
	}

	fmt.Println()

	// Save config
	ui.Divider()
	fmt.Println()

	if err := config.Save(cfg, configPath); err != nil {
		ui.Error(fmt.Sprintf("Failed to save config: %s", err))
		return err
	}

	ui.SuccessMsg(fmt.Sprintf("Created %s", configPath))
	fmt.Println()

	ui.OrderedSummary("Configuration", []ui.SummaryItem{
		{Label: "Provider", Value: cfg.Provider.Name},
		{Label: "Repository", Value: fmt.Sprintf("%s/%s", cfg.Provider.Owner, cfg.Provider.Repo)},
		{Label: "Main branch", Value: cfg.Branching.Main},
		{Label: "Merge method", Value: cfg.PR.MergeMethod},
		{Label: "Convention", Value: cfg.Commit.Convention},
	})

	ui.Info("Next steps:")
	nextToken := os.Getenv("GITHUB_TOKEN")
	if nextToken == "" {
		nextToken = os.Getenv("GH_TOKEN")
	}
	if nextToken == "" {
		fmt.Printf("    %s Set your token: export GITHUB_TOKEN=ghp_xxxxxxxxxxxx\n", ui.IconArrow)
	}
	fmt.Printf("    %s Start a feature: gflow start feature my-feature\n", ui.IconArrow)
	fmt.Printf("    %s Create a PR:     gflow pr\n", ui.IconArrow)
	fmt.Printf("    %s View dashboard:  gflow dash\n", ui.IconArrow)
	fmt.Println()

	return nil
}
