package cmd

import (
	"fmt"
	"strings"

	"github.com/abhinavcdev/gflow/internal/git"
	"github.com/abhinavcdev/gflow/internal/ui"
	"github.com/spf13/cobra"
)

var commitCmd = &cobra.Command{
	Use:   "commit [message]",
	Short: "Create a conventional commit with interactive type/scope selection",
	Long: `Stage changes and create a commit following your team's commit convention.
Interactively selects commit type, optional scope, and message.

Examples:
  gflow commit                          # Full interactive flow
  gflow commit "add user auth"          # Auto-detect type from branch
  gflow commit -t feat -m "add auth"    # Specify type and message
  gflow commit --all                    # Stage all before committing`,
	RunE: runCommit,
}

var (
	commitType    string
	commitScope   string
	commitMessage string
	commitAll     bool
	commitAmend   bool
)

func init() {
	commitCmd.Flags().StringVarP(&commitType, "type", "t", "", "commit type (feat, fix, docs, etc.)")
	commitCmd.Flags().StringVarP(&commitScope, "scope", "s", "", "commit scope")
	commitCmd.Flags().StringVarP(&commitMessage, "message", "m", "", "commit message")
	commitCmd.Flags().BoolVarP(&commitAll, "all", "a", false, "stage all changes before committing")
	commitCmd.Flags().BoolVar(&commitAmend, "amend", false, "amend the previous commit")
}

func runCommit(cmd *cobra.Command, args []string) error {
	g := git.NewFromCwd()

	if !g.IsRepo() {
		ui.Error("Not a git repository")
		return fmt.Errorf("not a git repository")
	}

	// Stage all if requested
	if commitAll {
		if err := g.AddAll(); err != nil {
			ui.StepFail("Failed to stage changes")
			return err
		}
		ui.StepDone("Staged all changes")
	}

	// Check for staged changes
	if !g.HasStagedChanges() && !commitAmend {
		if g.HasChanges() {
			ui.Warn("No staged changes. Use --all to stage everything, or 'git add' first.")
			status, _ := g.Status()
			fmt.Println(ui.MutedStyle.Render("    " + strings.ReplaceAll(status, "\n", "\n    ")))
			fmt.Println()

			stageAll, err := ui.PromptConfirm("Stage all changes?", true)
			if err != nil {
				return err
			}
			if stageAll {
				if err := g.AddAll(); err != nil {
					return err
				}
				ui.StepDone("Staged all changes")
			} else {
				return nil
			}
		} else {
			ui.Info("Nothing to commit — working tree clean")
			return nil
		}
	}

	// Build commit message
	var fullMessage string

	if cfg.Commit.Convention == "none" {
		// No convention — just get a message
		if len(args) > 0 {
			fullMessage = strings.Join(args, " ")
		} else if commitMessage != "" {
			fullMessage = commitMessage
		} else {
			var err error
			fullMessage, err = ui.PromptInputRequired("Commit message", "")
			if err != nil {
				return err
			}
		}
	} else {
		// Conventional commit flow
		cType := commitType
		if cType == "" {
			// Auto-detect from branch type
			branch, _ := g.CurrentBranch()
			branchType, _ := parseBranchName(branch)
			defaultType := mapBranchToCommitType(branchType)

			types := cfg.Commit.Types
			if len(types) == 0 {
				types = []string{"feat", "fix", "docs", "style", "refactor", "perf", "test", "build", "ci", "chore", "revert"}
			}

			// Put default type first
			orderedTypes := []string{}
			for _, t := range types {
				if t == defaultType {
					orderedTypes = append([]string{t}, orderedTypes...)
				} else {
					orderedTypes = append(orderedTypes, t)
				}
			}

			// Add descriptions
			typeLabels := make([]string, len(orderedTypes))
			for i, t := range orderedTypes {
				typeLabels[i] = fmt.Sprintf("%-10s %s", t, commitTypeDescription(t))
			}

			idx, _, err := ui.PromptSelect("Commit type", typeLabels)
			if err != nil {
				return err
			}
			cType = orderedTypes[idx]
		}

		// Scope
		cScope := commitScope
		if cScope == "" && (cfg.Commit.RequireScope || cfg.Commit.Convention == "angular") {
			if len(cfg.Commit.Scopes) > 0 {
				scopeOptions := append([]string{"(none)"}, cfg.Commit.Scopes...)
				idx, _, err := ui.PromptSelect("Scope", scopeOptions)
				if err != nil {
					return err
				}
				if idx > 0 {
					cScope = cfg.Commit.Scopes[idx-1]
				}
			} else {
				cScope, _ = ui.PromptInput("Scope (optional)", "")
			}
		}

		// Message
		msg := commitMessage
		if msg == "" && len(args) > 0 {
			msg = strings.Join(args, " ")
		}
		if msg == "" {
			var err error
			msg, err = ui.PromptInputRequired("Description", "")
			if err != nil {
				return err
			}
		}

		// Build full message
		if cScope != "" {
			fullMessage = fmt.Sprintf("%s(%s): %s", cType, cScope, msg)
		} else {
			fullMessage = fmt.Sprintf("%s: %s", cType, msg)
		}
	}

	// Run pre-commit hooks
	if len(cfg.Hooks.PreCommit) > 0 {
		ui.Info("Running pre-commit hooks...")
		for _, hook := range cfg.Hooks.PreCommit {
			if err := runHook(hook); err != nil {
				ui.StepFail(fmt.Sprintf("Hook failed: %s", hook))
				return err
			}
		}
		ui.StepDone("Pre-commit hooks passed")
	}

	// Commit
	s := ui.StartSpinner("Committing...")
	var err error
	if commitAmend {
		err = g.RunSilent("commit", "--amend", "-m", fullMessage)
	} else {
		err = g.Commit(fullMessage)
	}
	if err != nil {
		ui.StopSpinnerFail(s, "Failed to commit")
		return err
	}

	action := "Committed"
	if commitAmend {
		action = "Amended"
	}
	ui.StopSpinner(s, fmt.Sprintf("%s: %s", action, fullMessage))

	// Show commit info
	hash, _ := g.LastCommitHash()
	if hash != "" {
		ui.Detail("Hash", hash)
	}

	fmt.Println()
	return nil
}

func commitTypeDescription(t string) string {
	descriptions := map[string]string{
		"feat":     "A new feature",
		"fix":      "A bug fix",
		"docs":     "Documentation only changes",
		"style":    "Code style (formatting, semicolons, etc.)",
		"refactor": "Code change that neither fixes a bug nor adds a feature",
		"perf":     "A code change that improves performance",
		"test":     "Adding missing or correcting existing tests",
		"build":    "Changes to the build system or dependencies",
		"ci":       "Changes to CI configuration files and scripts",
		"chore":    "Other changes that don't modify src or test files",
		"revert":   "Reverts a previous commit",
	}
	if d, ok := descriptions[t]; ok {
		return ui.MutedStyle.Render(d)
	}
	return ""
}
