package cmd

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/abhinavcdev/gflow/internal/git"
	"github.com/abhinavcdev/gflow/internal/provider"
	"github.com/abhinavcdev/gflow/internal/ui"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var dashCmd = &cobra.Command{
	Use:     "dash",
	Aliases: []string{"dashboard", "d"},
	Short:   "Real-time workflow dashboard — your entire project at a glance",
	Long: `Launches a beautiful terminal dashboard that fetches everything in parallel
using Go's concurrency — branch state, open PRs, recent commits, sync status,
working tree, and more — all in one view.

This is the fastest way to understand where your project stands.

Examples:
  gflow dash
  gflow d`,
	RunE: runDash,
}

// dashData holds all dashboard state, fetched concurrently
type dashData struct {
	mu sync.Mutex

	// Git state
	currentBranch string
	branchType    string
	branchName    string
	baseBranch    string
	hasChanges    bool
	staged        int
	modified      int
	untracked     int
	commitsAhead  int
	commitsBehind int
	lastCommit    string
	lastHash      string
	recentCommits []commitEntry
	branches      []string
	stashCount    int

	// PR state
	currentPR    *provider.PullRequest
	openPRs      []*provider.PullRequest
	providerOk   bool
	providerName string

	// Timing
	fetchTime time.Duration

	// Errors (non-fatal)
	errors []string
}

type commitEntry struct {
	hash    string
	subject string
	author  string
	relTime string
}

// Styles for the dashboard
var (
	dashHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ui.Primary).
			MarginBottom(0)

	dashSection = lipgloss.NewStyle().
			Bold(true).
			Foreground(ui.Secondary)

	dashSuccess = lipgloss.NewStyle().Foreground(ui.Success)
	dashWarn    = lipgloss.NewStyle().Foreground(ui.Warning)
	dashError   = lipgloss.NewStyle().Foreground(ui.Danger)
	dashMuted   = lipgloss.NewStyle().Foreground(ui.Muted)
	dashBold    = lipgloss.NewStyle().Bold(true)
)

func runDash(cmd *cobra.Command, args []string) error {
	g := git.NewFromCwd()

	if !g.IsRepo() {
		ui.Error("Not a git repository")
		return fmt.Errorf("not a git repository")
	}

	start := time.Now()

	// Fetch all data concurrently using goroutines
	data := &dashData{}
	var wg sync.WaitGroup

	// --- Goroutine 1: Branch & working tree state ---
	wg.Add(1)
	go func() {
		defer wg.Done()
		data.mu.Lock()
		data.currentBranch, _ = g.CurrentBranch()
		data.branchType, data.branchName = parseBranchName(data.currentBranch)
		data.baseBranch = detectBaseBranch(data.currentBranch)
		data.hasChanges = g.HasChanges()
		data.mu.Unlock()

		if data.hasChanges {
			status, _ := g.Status()
			s, m, u := parseStatusCounts(status)
			data.mu.Lock()
			data.staged = s
			data.modified = m
			data.untracked = u
			data.mu.Unlock()
		}
	}()

	// --- Goroutine 2: Commit history & ahead/behind ---
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Last commit
		msg, _ := g.LastCommitMessage()
		hash, _ := g.LastCommitHash()
		data.mu.Lock()
		data.lastCommit = msg
		data.lastHash = hash
		data.mu.Unlock()

		// Wait for branch info
		for data.currentBranch == "" {
			time.Sleep(time.Millisecond)
		}

		// Commits ahead
		count, err := g.CommitCount(data.baseBranch)
		if err == nil {
			data.mu.Lock()
			data.commitsAhead = count
			data.mu.Unlock()
		}

		// Recent commits
		logOut, err := g.Log(8, "%h||%s||%an||%ar")
		if err == nil {
			var commits []commitEntry
			for _, line := range strings.Split(logOut, "\n") {
				parts := strings.SplitN(strings.TrimSpace(line), "||", 4)
				if len(parts) == 4 {
					commits = append(commits, commitEntry{
						hash:    parts[0],
						subject: parts[1],
						author:  parts[2],
						relTime: parts[3],
					})
				}
			}
			data.mu.Lock()
			data.recentCommits = commits
			data.mu.Unlock()
		}

		// Stash count
		stashOut, err := g.Run("stash", "list")
		if err == nil && stashOut != "" {
			count := len(strings.Split(strings.TrimSpace(stashOut), "\n"))
			data.mu.Lock()
			data.stashCount = count
			data.mu.Unlock()
		}
	}()

	// --- Goroutine 3: Branch list ---
	wg.Add(1)
	go func() {
		defer wg.Done()
		out, err := g.Run("branch", "--format=%(refname:short)")
		if err == nil {
			var branches []string
			for _, b := range strings.Split(out, "\n") {
				b = strings.TrimSpace(b)
				if b != "" {
					branches = append(branches, b)
				}
			}
			data.mu.Lock()
			data.branches = branches
			data.mu.Unlock()
		}
	}()

	// --- Goroutine 4: Fetch from remote (background) ---
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = g.Fetch()

		// Check behind count after fetch
		for data.currentBranch == "" {
			time.Sleep(time.Millisecond)
		}
		behindOut, err := g.Run("rev-list", "--count", data.currentBranch+"..origin/"+data.baseBranch)
		if err == nil {
			var behind int
			_, _ = fmt.Sscanf(strings.TrimSpace(behindOut), "%d", &behind)
			data.mu.Lock()
			data.commitsBehind = behind
			data.mu.Unlock()
		}
	}()

	// --- Goroutine 5: Provider / PR data ---
	wg.Add(1)
	go func() {
		defer wg.Done()
		data.mu.Lock()
		data.providerName = cfg.Provider.Name
		data.mu.Unlock()

		p, err := provider.New(cfg)
		if err != nil {
			data.mu.Lock()
			data.errors = append(data.errors, "Provider: "+err.Error())
			data.mu.Unlock()
			return
		}

		data.mu.Lock()
		data.providerOk = true
		data.mu.Unlock()

		// Fetch open PRs and current branch PR concurrently
		var prWg sync.WaitGroup

		prWg.Add(1)
		go func() {
			defer prWg.Done()
			prs, err := p.ListPRs()
			if err == nil {
				data.mu.Lock()
				data.openPRs = prs
				data.mu.Unlock()
			}
		}()

		prWg.Add(1)
		go func() {
			defer prWg.Done()
			for data.currentBranch == "" {
				time.Sleep(time.Millisecond)
			}
			pr, err := p.GetPRForBranch(data.currentBranch)
			if err == nil {
				data.mu.Lock()
				data.currentPR = pr
				data.mu.Unlock()
			}
		}()

		prWg.Wait()
	}()

	// Wait for all goroutines
	wg.Wait()
	data.fetchTime = time.Since(start)

	// Render the dashboard
	renderDashboard(data)

	return nil
}

func renderDashboard(d *dashData) {
	width := 62

	// Header
	header := dashHeader.Render(
		"         __ _                \n   __ _ / _| | _____      __\n  / _` | |_| |/ _ \\ \\ /\\ / /\n | (_| |  _| | (_) \\ V  V / \n  \\__, |_| |_|\\___/ \\_/\\_/  \n  |___/",
	)
	subtitle := dashMuted.Render("  opinionated git workflow CLI")
	timeStr := dashMuted.Render(fmt.Sprintf("  fetched in %dms using %d parallel goroutines", d.fetchTime.Milliseconds(), 5))

	fmt.Println(header)
	fmt.Println(subtitle)
	fmt.Println(timeStr)
	fmt.Println()

	// ─── BRANCH ─────────────────────────────────────────
	fmt.Println(dashSection.Render("  ─── BRANCH " + strings.Repeat("─", width-13)))
	fmt.Println()

	branchIcon := "●"
	switch d.branchType {
	case "feature", "feat":
		branchIcon = dashSuccess.Render("✨")
	case "bugfix", "fix":
		branchIcon = dashError.Render("🐛")
	case "hotfix":
		branchIcon = dashError.Render("🔥")
	case "release":
		branchIcon = dashWarn.Render("🏷 ")
	default:
		if d.currentBranch == "main" || d.currentBranch == "master" {
			branchIcon = dashBold.Render("🔒")
		}
	}

	fmt.Printf("  %s %s", branchIcon, dashBold.Render(d.currentBranch))
	if d.branchName != d.currentBranch {
		fmt.Printf("  %s", dashMuted.Render("("+d.branchType+")"))
	}
	fmt.Println()
	fmt.Printf("    base: %s\n", dashMuted.Render(d.baseBranch))

	// Ahead/behind
	syncParts := []string{}
	if d.commitsAhead > 0 {
		syncParts = append(syncParts, dashSuccess.Render(fmt.Sprintf("↑%d ahead", d.commitsAhead)))
	}
	if d.commitsBehind > 0 {
		syncParts = append(syncParts, dashWarn.Render(fmt.Sprintf("↓%d behind", d.commitsBehind)))
	}
	if len(syncParts) == 0 {
		syncParts = append(syncParts, dashSuccess.Render("✓ in sync"))
	}
	fmt.Printf("    sync: %s\n", strings.Join(syncParts, "  "))
	fmt.Println()

	// ─── WORKING TREE ───────────────────────────────────
	fmt.Println(dashSection.Render("  ─── WORKING TREE " + strings.Repeat("─", width-19)))
	fmt.Println()

	if d.hasChanges {
		parts := []string{}
		if d.staged > 0 {
			parts = append(parts, dashSuccess.Render(fmt.Sprintf("%d staged", d.staged)))
		}
		if d.modified > 0 {
			parts = append(parts, dashWarn.Render(fmt.Sprintf("%d modified", d.modified)))
		}
		if d.untracked > 0 {
			parts = append(parts, dashMuted.Render(fmt.Sprintf("%d untracked", d.untracked)))
		}
		fmt.Printf("  %s  %s\n", dashWarn.Render("●"), strings.Join(parts, "  "))
	} else {
		fmt.Printf("  %s  %s\n", dashSuccess.Render("●"), dashSuccess.Render("clean"))
	}

	if d.stashCount > 0 {
		fmt.Printf("  %s  %s\n", dashMuted.Render("📦"), dashMuted.Render(fmt.Sprintf("%d stash(es)", d.stashCount)))
	}
	fmt.Println()

	// ─── PULL REQUEST ───────────────────────────────────
	fmt.Println(dashSection.Render("  ─── PULL REQUEST " + strings.Repeat("─", width-19)))
	fmt.Println()

	if d.currentPR != nil {
		pr := d.currentPR
		stateIcon := dashSuccess.Render("●")
		stateLabel := "Open"
		if pr.Draft {
			stateIcon = dashMuted.Render("◌")
			stateLabel = "Draft"
		}
		fmt.Printf("  %s  #%d %s  %s\n", stateIcon, pr.Number, dashBold.Render(pr.Title), dashMuted.Render(stateLabel))
		fmt.Printf("       %s\n", dashMuted.Render(pr.URL))
		if len(pr.Reviewers) > 0 {
			fmt.Printf("       reviewers: %s\n", strings.Join(pr.Reviewers, ", "))
		}
		if len(pr.Labels) > 0 {
			fmt.Printf("       labels: %s\n", dashMuted.Render(strings.Join(pr.Labels, ", ")))
		}
	} else if d.providerOk {
		if d.currentBranch == "main" || d.currentBranch == "master" || d.currentBranch == "develop" {
			fmt.Printf("  %s  %s\n", dashMuted.Render("─"), dashMuted.Render("on base branch"))
		} else {
			fmt.Printf("  %s  %s  →  %s\n", dashWarn.Render("○"), dashMuted.Render("no PR yet"), dashBold.Render("gflow pr"))
		}
	} else {
		fmt.Printf("  %s  %s\n", dashMuted.Render("─"), dashMuted.Render("provider not connected"))
	}
	fmt.Println()

	// ─── OPEN PRs ───────────────────────────────────────
	if d.providerOk && len(d.openPRs) > 0 {
		fmt.Println(dashSection.Render(fmt.Sprintf("  ─── OPEN PRs (%d) ", len(d.openPRs)) + strings.Repeat("─", width-18-len(fmt.Sprintf("%d", len(d.openPRs))))))
		fmt.Println()

		limit := 5
		if len(d.openPRs) < limit {
			limit = len(d.openPRs)
		}
		for _, pr := range d.openPRs[:limit] {
			icon := dashSuccess.Render("●")
			if pr.Draft {
				icon = dashMuted.Render("◌")
			}
			marker := ""
			if pr.Head == d.currentBranch {
				marker = dashSuccess.Render(" ←")
			}
			title := pr.Title
			if len(title) > 42 {
				title = title[:39] + "..."
			}
			fmt.Printf("  %s  #%-4d %s%s\n", icon, pr.Number, title, marker)
			fmt.Printf("          %s\n", dashMuted.Render(pr.Head+" → "+pr.Base))
		}
		if len(d.openPRs) > limit {
			fmt.Printf("  %s\n", dashMuted.Render(fmt.Sprintf("  ... and %d more", len(d.openPRs)-limit)))
		}
		fmt.Println()
	}

	// ─── RECENT COMMITS ─────────────────────────────────
	if len(d.recentCommits) > 0 {
		fmt.Println(dashSection.Render("  ─── RECENT COMMITS " + strings.Repeat("─", width-21)))
		fmt.Println()

		limit := 6
		if len(d.recentCommits) < limit {
			limit = len(d.recentCommits)
		}
		for _, c := range d.recentCommits[:limit] {
			var icon string
			cType := extractCommitType(c.subject)
			switch cType {
			case "feat":
				icon = "✨"
			case "fix":
				icon = "🐛"
			case "docs":
				icon = "📖"
			case "refactor":
				icon = "♻️ "
			case "test":
				icon = "🧪"
			case "ci", "build":
				icon = "⚙️ "
			case "chore":
				icon = "🔧"
			case "perf":
				icon = "⚡"
			default:
				icon = "  "
			}
			subject := c.subject
			if len(subject) > 45 {
				subject = subject[:42] + "..."
			}
			fmt.Printf("  %s %s %s  %s\n",
				icon,
				dashWarn.Render(c.hash),
				subject,
				dashMuted.Render(c.relTime),
			)
		}
		fmt.Println()
	}

	// ─── BRANCHES ───────────────────────────────────────
	if len(d.branches) > 1 {
		fmt.Println(dashSection.Render(fmt.Sprintf("  ─── BRANCHES (%d) ", len(d.branches)) + strings.Repeat("─", width-20-len(fmt.Sprintf("%d", len(d.branches))))))
		fmt.Println()

		for _, b := range d.branches {
			marker := "  "
			if b == d.currentBranch {
				marker = dashSuccess.Render("→ ")
			}
			bType, _ := parseBranchName(b)
			icon := " "
			switch bType {
			case "feature", "feat":
				icon = "✨"
			case "bugfix", "fix":
				icon = "🐛"
			case "hotfix":
				icon = "🔥"
			case "release":
				icon = "🏷 "
			default:
				if b == "main" || b == "master" || b == "develop" {
					icon = "🔒"
				}
			}
			fmt.Printf("  %s%s %s\n", marker, icon, b)
		}
		fmt.Println()
	}

	// ─── QUICK ACTIONS ──────────────────────────────────
	fmt.Println(dashSection.Render("  ─── QUICK ACTIONS " + strings.Repeat("─", width-20)))
	fmt.Println()

	actions := []struct {
		cmd  string
		desc string
		show bool
	}{
		{"gflow start feature <name>", "start a new feature branch", d.currentBranch == "main" || d.currentBranch == "master"},
		{"gflow commit", "conventional commit your changes", d.hasChanges},
		{"gflow pr", "open a pull request", d.currentPR == nil && d.currentBranch != "main" && d.currentBranch != "master"},
		{"gflow pr view", "view your open PR", d.currentPR != nil},
		{"gflow finish", "merge PR & clean up", d.currentPR != nil},
		{"gflow sync", "rebase on " + d.baseBranch, d.commitsBehind > 0},
		{"gflow clean", "prune merged branches", len(d.branches) > 3},
	}

	shown := 0
	for _, a := range actions {
		if a.show && shown < 4 {
			fmt.Printf("  %s  %s\n", dashBold.Render(fmt.Sprintf("%-35s", a.cmd)), dashMuted.Render(a.desc))
			shown++
		}
	}
	if shown == 0 {
		fmt.Printf("  %s\n", dashMuted.Render("all good — nothing urgent"))
	}
	fmt.Println()

	// Footer
	fmt.Println(dashMuted.Render(fmt.Sprintf("  %s • %s/%s • %dms",
		cfg.Provider.Name,
		cfg.Provider.Owner,
		cfg.Provider.Repo,
		d.fetchTime.Milliseconds(),
	)))
	fmt.Println()
}

func parseStatusCounts(status string) (staged, modified, untracked int) {
	for _, l := range strings.Split(status, "\n") {
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
	return
}
