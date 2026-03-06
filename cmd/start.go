package cmd

import (
	"fmt"
	"strings"

	"github.com/abhinavcdev/gflow/internal/git"
	"github.com/abhinavcdev/gflow/internal/ui"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start <type> <name>",
	Short: "Start a new branch (feature, bugfix, hotfix, release, or custom strategy)",
	Long: `Create a new branch following your team's naming conventions.

Built-in types: feature, bugfix, hotfix, release
Custom strategies defined in .gflow.yml are also available.

Examples:
  gflow start feature user-auth
  gflow start bugfix login-redirect
  gflow start hotfix critical-fix
  gflow start release 1.2.0
  gflow start my-custom-strategy task-name`,
	Args: cobra.MinimumNArgs(1),
	RunE: runStart,
}

var (
	startBase   string
	startFromCurrent bool
)

func init() {
	startCmd.Flags().StringVarP(&startBase, "base", "b", "", "base branch to create from (overrides strategy default)")
	startCmd.Flags().BoolVar(&startFromCurrent, "from-current", false, "create from current branch instead of base")
}

func runStart(cmd *cobra.Command, args []string) error {
	strategyName := args[0]
	var name string

	if len(args) > 1 {
		name = strings.Join(args[1:], "-")
	}

	g := git.NewFromCwd()

	if !g.IsRepo() {
		ui.Error("Not a git repository")
		return fmt.Errorf("not a git repository")
	}

	// Find strategy
	strategy, found := cfg.GetStrategy(strategyName)
	if !found {
		ui.Errorf("Unknown branch type or strategy: %s", strategyName)
		ui.Info("Available types: feature, bugfix, hotfix, release")
		if cfg.Strategies != nil {
			var custom []string
			for k := range cfg.Strategies {
				custom = append(custom, k)
			}
			if len(custom) > 0 {
				ui.Info(fmt.Sprintf("Custom strategies: %s", strings.Join(custom, ", ")))
			}
		}
		return fmt.Errorf("unknown strategy: %s", strategyName)
	}

	// Prompt for name if not provided
	if name == "" {
		var err error
		name, err = ui.PromptInputRequired("Branch name (slug)", "")
		if err != nil {
			return err
		}
	}

	// Sanitize branch name
	name = sanitizeBranchName(name)
	branchName := strategy.Prefix + name

	ui.Title(fmt.Sprintf("  Starting %s: %s", strategyName, name))
	fmt.Println()

	// Determine base branch
	baseBranch := strategy.BaseBranch
	if startBase != "" {
		baseBranch = startBase
	}

	// Check if branch already exists
	if g.BranchExists(branchName) {
		ui.Errorf("Branch %s already exists", branchName)

		switchTo, err := ui.PromptConfirm(fmt.Sprintf("Switch to %s?", branchName), true)
		if err != nil {
			return err
		}
		if switchTo {
			if err := g.Checkout(branchName); err != nil {
				ui.StepFail(fmt.Sprintf("Failed to switch to %s", branchName))
				return err
			}
			ui.StepDone(fmt.Sprintf("Switched to %s", branchName))
		}
		return nil
	}

	// Run pre-create hooks
	if len(strategy.Hooks.PreCreate) > 0 {
		ui.Step(1, "Running pre-create hooks...")
		for _, hook := range strategy.Hooks.PreCreate {
			if err := runHook(hook); err != nil {
				ui.StepFail(fmt.Sprintf("Hook failed: %s", hook))
				return err
			}
		}
		ui.StepDone("Pre-create hooks passed")
	}

	// Fetch latest
	s := ui.StartSpinner("Fetching latest changes...")
	if err := g.Fetch(); err != nil {
		ui.StopSpinnerFail(s, "Failed to fetch")
		ui.Warn("Continuing without fetch (you may be offline)")
	} else {
		ui.StopSpinner(s, "Fetched latest changes")
	}

	if startFromCurrent {
		// Create from current branch
		s = ui.StartSpinner(fmt.Sprintf("Creating branch %s from current HEAD...", branchName))
		if err := g.CreateBranchFromCurrent(branchName); err != nil {
			ui.StopSpinnerFail(s, "Failed to create branch")
			return err
		}
		ui.StopSpinner(s, fmt.Sprintf("Created %s from current branch", branchName))
	} else {
		// Switch to base and update
		s = ui.StartSpinner(fmt.Sprintf("Updating %s...", baseBranch))
		if err := g.Checkout(baseBranch); err != nil {
			ui.StopSpinnerFail(s, fmt.Sprintf("Failed to checkout %s", baseBranch))
			return err
		}
		if err := g.PullRebase(); err != nil {
			ui.Warn(fmt.Sprintf("Failed to pull %s (continuing anyway)", baseBranch))
		}
		ui.StopSpinner(s, fmt.Sprintf("Updated %s", baseBranch))

		// Create the new branch
		s = ui.StartSpinner(fmt.Sprintf("Creating branch %s...", branchName))
		if err := g.CreateBranchFromCurrent(branchName); err != nil {
			ui.StopSpinnerFail(s, "Failed to create branch")
			return err
		}
		ui.StopSpinner(s, fmt.Sprintf("Created %s", branchName))
	}

	// Push to remote
	s = ui.StartSpinner("Pushing to remote...")
	if err := g.PushSetUpstream(); err != nil {
		ui.StopSpinnerFail(s, "Failed to push (you can push manually later)")
	} else {
		ui.StopSpinner(s, "Pushed to remote")
	}

	// Run post-create hooks
	if len(strategy.Hooks.PostCreate) > 0 {
		for _, hook := range strategy.Hooks.PostCreate {
			if err := runHook(hook); err != nil {
				ui.Warn(fmt.Sprintf("Post-create hook failed: %s", hook))
			}
		}
	}

	fmt.Println()
	ui.SuccessMsg(fmt.Sprintf("Ready to work on %s", branchName))
	ui.Detail("Branch", branchName)
	ui.Detail("Base", baseBranch)
	ui.Detail("Type", strategyName)
	fmt.Println()
	ui.Info("Next: make changes, then run 'gflow pr' to open a pull request")
	fmt.Println()

	return nil
}

func sanitizeBranchName(name string) string {
	name = strings.ToLower(name)
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '/' || r == '.' {
			return r
		}
		return '-'
	}, name)
	// Remove consecutive dashes
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}
	name = strings.Trim(name, "-")
	return name
}

func runHook(command string) error {
	// Execute shell command
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil
	}
	cmd := newExecCommand(parts[0], parts[1:]...)
	return cmd.Run()
}
