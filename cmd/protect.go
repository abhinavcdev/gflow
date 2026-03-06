package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/abhinavcdev/gflow/internal/ui"
	"github.com/spf13/cobra"
)

var protectCmd = &cobra.Command{
	Use:   "protect [branch]",
	Short: "Set up GitHub branch protection rules from your .gflow.yml",
	Long: `Configures GitHub branch protection rules via the API using your token.
Sets up required status checks, PR reviews, and merge restrictions
based on your .gflow.yml and CI workflow.

Requires a GitHub token with admin/repo permissions.

Examples:
  gflow protect                # Protect main branch with CI checks
  gflow protect main           # Explicit branch
  gflow protect --checks       # Show what would be configured (dry run)`,
	RunE: runProtect,
}

var (
	protectDryRun     bool
	protectRequirePR  bool
	protectDismissStale bool
	protectEnforce    bool
)

func init() {
	protectCmd.Flags().BoolVar(&protectDryRun, "checks", false, "dry run — show what would be configured")
	protectCmd.Flags().BoolVar(&protectRequirePR, "require-pr", true, "require PR before merging")
	protectCmd.Flags().BoolVar(&protectDismissStale, "dismiss-stale", true, "dismiss stale PR reviews on new pushes")
	protectCmd.Flags().BoolVar(&protectEnforce, "enforce-admins", false, "enforce rules for admins too")
}

func runProtect(cmd *cobra.Command, args []string) error {
	ui.Title("  Protect Branch")
	fmt.Println()

	if cfg.Provider.Name != "github" {
		ui.Error("Branch protection via API is currently supported for GitHub only")
		return fmt.Errorf("unsupported provider: %s", cfg.Provider.Name)
	}

	token := cfg.GetToken()
	if token == "" {
		ui.Error("No GitHub token found")
		fmt.Println()
		fmt.Println("  Set your token:")
		fmt.Println(ui.BoldStyle.Render("    export GITHUB_TOKEN=ghp_xxxxxxxxxxxx"))
		fmt.Println()
		fmt.Println(ui.MutedStyle.Render("  Token needs admin:repo scope for branch protection."))
		return fmt.Errorf("no token")
	}

	// Determine branch to protect
	branch := cfg.Branching.Main
	if len(args) > 0 {
		branch = args[0]
	}

	// Detect CI check names from .gflow.yml hooks and common CI jobs
	requiredChecks := detectRequiredChecks()

	// Build the protection payload
	protection := buildProtectionPayload(requiredChecks)

	// Show summary
	ui.Detail("Repository", fmt.Sprintf("%s/%s", cfg.Provider.Owner, cfg.Provider.Repo))
	ui.Detail("Branch", branch)
	fmt.Println()

	fmt.Println(ui.BoldStyle.Render("  Rules to apply:"))
	fmt.Println()
	fmt.Printf("    %s Require PR before merging: %s\n", ui.IconDot, formatBool(protectRequirePR))
	fmt.Printf("    %s Dismiss stale reviews: %s\n", ui.IconDot, formatBool(protectDismissStale))
	fmt.Printf("    %s Enforce for admins: %s\n", ui.IconDot, formatBool(protectEnforce))
	fmt.Printf("    %s Required status checks:\n", ui.IconDot)
	if len(requiredChecks) > 0 {
		for _, c := range requiredChecks {
			fmt.Printf("        %s %s\n", ui.SuccessStyle.Render("✓"), c)
		}
	} else {
		fmt.Printf("        %s\n", ui.MutedStyle.Render("(none detected — add CI jobs to enforce)"))
	}
	fmt.Printf("    %s Require branch to be up-to-date: %s\n", ui.IconDot, formatBool(len(requiredChecks) > 0))
	fmt.Printf("    %s Delete branch on merge: %s\n", ui.IconDot, formatBool(cfg.PR.DeleteBranch))
	fmt.Println()

	if protectDryRun {
		ui.Info("Dry run — no changes made")
		fmt.Println()
		prettyJSON, _ := json.MarshalIndent(protection, "  ", "  ")
		fmt.Println(ui.MutedStyle.Render("  " + string(prettyJSON)))
		fmt.Println()
		return nil
	}

	// Confirm
	proceed, err := ui.PromptConfirm(fmt.Sprintf("Apply branch protection to %s?", branch), true)
	if err != nil || !proceed {
		ui.Info("Aborted")
		return nil
	}

	fmt.Println()

	// Apply via GitHub API
	s := ui.StartSpinner("Applying branch protection rules...")

	apiURL := "https://api.github.com"
	if cfg.Provider.Host != "" {
		apiURL = fmt.Sprintf("https://%s/api/v3", cfg.Provider.Host)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/branches/%s/protection",
		apiURL, cfg.Provider.Owner, cfg.Provider.Repo, branch)

	body, err := json.Marshal(protection)
	if err != nil {
		ui.StopSpinnerFail(s, "Failed to build request")
		return err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		ui.StopSpinnerFail(s, "Failed to build request")
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ui.StopSpinnerFail(s, "Failed to connect to GitHub")
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		ui.StopSpinnerFail(s, "Failed to set branch protection")
		var ghErr struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(respBody, &ghErr)
		ui.Error(fmt.Sprintf("GitHub API error (%d): %s", resp.StatusCode, ghErr.Message))

		if resp.StatusCode == 403 {
			fmt.Println()
			ui.Info("Your token may need the 'admin:repo' scope.")
			fmt.Println("  Regenerate at: https://github.com/settings/tokens")
		}
		if resp.StatusCode == 404 {
			fmt.Println()
			ui.Info(fmt.Sprintf("Branch '%s' may not exist on remote, or you don't have admin access.", branch))
		}
		return fmt.Errorf("API error: %s", ghErr.Message)
	}

	ui.StopSpinner(s, "Branch protection applied!")

	// Also set delete-branch-on-merge repo setting if configured
	if cfg.PR.DeleteBranch {
		setDeleteBranchOnMerge(apiURL, token)
	}

	fmt.Println()
	ui.SuccessMsg(fmt.Sprintf("Branch '%s' is now protected", branch))
	ui.Detail("Required checks", strings.Join(requiredChecks, ", "))
	ui.Detail("Require PR", fmt.Sprintf("%v", protectRequirePR))
	ui.Detail("View rules", fmt.Sprintf("https://github.com/%s/%s/settings/branches", cfg.Provider.Owner, cfg.Provider.Repo))
	fmt.Println()

	return nil
}

func detectRequiredChecks() []string {
	checks := []string{}

	// Add standard CI job names that match our ci.yml
	checks = append(checks, "Test (ubuntu-latest, 1.24)")
	checks = append(checks, "Lint")

	return checks
}

func buildProtectionPayload(checks []string) map[string]interface{} {
	// Required status checks
	statusChecks := map[string]interface{}{
		"strict": len(checks) > 0, // require branch to be up-to-date
	}

	if len(checks) > 0 {
		checkObjs := []map[string]string{}
		for _, c := range checks {
			checkObjs = append(checkObjs, map[string]string{
				"context": c,
			})
		}
		statusChecks["checks"] = checkObjs
	}

	// Required PR reviews
	prReviews := map[string]interface{}{}
	if protectRequirePR {
		prReviews["dismiss_stale_reviews"] = protectDismissStale
		prReviews["require_code_owner_reviews"] = false
		prReviews["required_approving_review_count"] = 0
	}

	payload := map[string]interface{}{
		"required_status_checks":          statusChecks,
		"enforce_admins":                  protectEnforce,
		"required_pull_request_reviews":   prReviews,
		"restrictions":                    nil,
		"required_linear_history":         cfg.PR.MergeMethod == "squash" || cfg.PR.MergeMethod == "rebase",
		"allow_force_pushes":              false,
		"allow_deletions":                 false,
	}

	return payload
}

func setDeleteBranchOnMerge(apiURL, token string) {
	url := fmt.Sprintf("%s/repos/%s/%s", apiURL, cfg.Provider.Owner, cfg.Provider.Repo)

	body, _ := json.Marshal(map[string]interface{}{
		"delete_branch_on_merge": true,
	})

	req, err := http.NewRequest("PATCH", url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 400 {
		ui.StepDone("Auto-delete branches on merge enabled")
	}
}

func formatBool(b bool) string {
	if b {
		return ui.SuccessStyle.Render("yes")
	}
	return ui.MutedStyle.Render("no")
}
