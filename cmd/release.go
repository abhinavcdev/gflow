package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/abhinavcdev/gflow/internal/git"
	"github.com/abhinavcdev/gflow/internal/provider"
	"github.com/abhinavcdev/gflow/internal/ui"
	"github.com/spf13/cobra"
)

var releaseCmd = &cobra.Command{
	Use:   "release <version>",
	Short: "Create a release — tag, changelog, and publish",
	Long: `Automates the full release workflow:
1. Creates a release branch (if using git-flow)
2. Bumps version
3. Generates changelog from commits
4. Tags the release
5. Pushes tag and branch
6. Creates a GitHub/GitLab/Bitbucket release

Examples:
  gflow release 1.2.0          # Create release v1.2.0
  gflow release 1.2.0 --draft  # Create as draft release
  gflow release patch           # Auto-bump patch version
  gflow release minor           # Auto-bump minor version
  gflow release major           # Auto-bump major version`,
	Args: cobra.ExactArgs(1),
	RunE: runRelease,
}

var (
	releaseDraft   bool
	releasePre     bool
	releaseNoTag   bool
	releaseNoPush  bool
	releaseNotes   string
)

func init() {
	releaseCmd.Flags().BoolVar(&releaseDraft, "draft", false, "create as draft release")
	releaseCmd.Flags().BoolVar(&releasePre, "prerelease", false, "mark as prerelease")
	releaseCmd.Flags().BoolVar(&releaseNoTag, "no-tag", false, "skip creating git tag")
	releaseCmd.Flags().BoolVar(&releaseNoPush, "no-push", false, "skip pushing to remote")
	releaseCmd.Flags().StringVar(&releaseNotes, "notes", "", "release notes")
}

func runRelease(cmd *cobra.Command, args []string) error {
	version := args[0]

	g := git.NewFromCwd()

	if !g.IsRepo() {
		ui.Error("Not a git repository")
		return fmt.Errorf("not a git repository")
	}

	ui.Title("  Create Release")
	fmt.Println()

	// Handle semantic version bumps
	if version == "patch" || version == "minor" || version == "major" {
		lastTag, err := g.Run("describe", "--tags", "--abbrev=0")
		if err != nil {
			version = "0.1.0"
			ui.Info(fmt.Sprintf("No previous tags found, using %s", version))
		} else {
			version = bumpVersion(strings.TrimPrefix(lastTag, "v"), version)
			ui.Info(fmt.Sprintf("Bumping from %s to %s", lastTag, version))
		}
	}

	// Ensure version has v prefix for tag
	tagName := version
	if !strings.HasPrefix(tagName, "v") {
		tagName = "v" + tagName
	}

	currentBranch, err := g.CurrentBranch()
	if err != nil {
		ui.Error("Failed to determine current branch")
		return err
	}

	// Generate changelog
	var changelog string
	if releaseNotes != "" {
		changelog = releaseNotes
	} else {
		changelog = generateChangelog(g, tagName)
	}

	ui.OrderedSummary("Release", []ui.SummaryItem{
		{Label: "Version", Value: tagName},
		{Label: "Branch", Value: currentBranch},
		{Label: "Draft", Value: fmt.Sprintf("%v", releaseDraft)},
		{Label: "Prerelease", Value: fmt.Sprintf("%v", releasePre)},
	})

	// Confirm
	proceed, err := ui.PromptConfirm("Proceed with release?", true)
	if err != nil || !proceed {
		ui.Info("Aborted")
		return nil
	}

	fmt.Println()

	// Create tag
	if !releaseNoTag {
		s := ui.StartSpinner(fmt.Sprintf("Creating tag %s...", tagName))
		if err := g.Tag(tagName, fmt.Sprintf("Release %s", tagName)); err != nil {
			ui.StopSpinnerFail(s, "Failed to create tag")
			return err
		}
		ui.StopSpinner(s, fmt.Sprintf("Created tag %s", tagName))
	}

	// Push
	if !releaseNoPush {
		s := ui.StartSpinner("Pushing to remote...")
		if err := g.Push(); err != nil {
			// Might need upstream
			g.PushSetUpstream()
		}
		if !releaseNoTag {
			if err := g.PushTag(tagName); err != nil {
				ui.StopSpinnerFail(s, "Failed to push tag")
				return err
			}
		}
		ui.StopSpinner(s, "Pushed to remote")
	}

	// Create release on provider
	p, err := provider.New(cfg)
	if err != nil {
		ui.Warn("Cannot connect to provider — tag was created locally")
		ui.Info("Create the release manually on your provider's web UI")
	} else {
		s := ui.StartSpinner("Creating release...")
		rel, err := p.CreateRelease(provider.ReleaseCreateOptions{
			TagName:    tagName,
			Name:       fmt.Sprintf("Release %s", tagName),
			Body:       changelog,
			Draft:      releaseDraft,
			Prerelease: releasePre,
			Target:     currentBranch,
		})
		if err != nil {
			ui.StopSpinnerFail(s, "Failed to create release")
			ui.Warn("The tag was pushed — create the release manually")
		} else {
			ui.StopSpinner(s, "Release created!")
			ui.Detail("URL", rel.URL)
		}
	}

	fmt.Println()
	ui.SuccessMsg(fmt.Sprintf("Released %s", tagName))
	fmt.Println()

	return nil
}

func generateChangelog(g *git.Git, tagName string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", tagName))
	sb.WriteString(fmt.Sprintf("Released on %s\n\n", time.Now().Format("2006-01-02")))

	// Try to get commits since last tag
	lastTag, err := g.Run("describe", "--tags", "--abbrev=0", "HEAD^")
	var commits string
	if err == nil {
		commits, _ = g.LogBetween(lastTag, "HEAD", "%s")
	} else {
		commits, _ = g.Log(50, "%s")
	}

	if commits == "" {
		sb.WriteString("No changes recorded.\n")
		return sb.String()
	}

	// Categorize commits
	features := []string{}
	fixes := []string{}
	other := []string{}

	for _, line := range strings.Split(commits, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "feat"):
			features = append(features, cleanCommitLine(line))
		case strings.HasPrefix(line, "fix"):
			fixes = append(fixes, cleanCommitLine(line))
		default:
			other = append(other, line)
		}
	}

	if len(features) > 0 {
		sb.WriteString("## ✨ Features\n\n")
		for _, f := range features {
			sb.WriteString(fmt.Sprintf("- %s\n", f))
		}
		sb.WriteString("\n")
	}

	if len(fixes) > 0 {
		sb.WriteString("## 🐛 Bug Fixes\n\n")
		for _, f := range fixes {
			sb.WriteString(fmt.Sprintf("- %s\n", f))
		}
		sb.WriteString("\n")
	}

	if len(other) > 0 {
		sb.WriteString("## 📦 Other Changes\n\n")
		for _, o := range other {
			sb.WriteString(fmt.Sprintf("- %s\n", o))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func cleanCommitLine(line string) string {
	// Remove conventional commit prefix
	if idx := strings.Index(line, ":"); idx > 0 && idx < 20 {
		return strings.TrimSpace(line[idx+1:])
	}
	return line
}

func bumpVersion(current, bump string) string {
	parts := strings.Split(current, ".")
	if len(parts) != 3 {
		return "0.1.0"
	}

	major, minor, patch := 0, 0, 0
	fmt.Sscanf(parts[0], "%d", &major)
	fmt.Sscanf(parts[1], "%d", &minor)
	fmt.Sscanf(parts[2], "%d", &patch)

	switch bump {
	case "major":
		major++
		minor = 0
		patch = 0
	case "minor":
		minor++
		patch = 0
	case "patch":
		patch++
	}

	return fmt.Sprintf("%d.%d.%d", major, minor, patch)
}
