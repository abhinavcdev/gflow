package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// Git wraps git CLI operations
type Git struct {
	dir string
}

// New creates a new Git instance for the given directory
func New(dir string) *Git {
	return &Git{dir: dir}
}

// NewFromCwd creates a new Git instance for the current working directory
func NewFromCwd() *Git {
	return &Git{}
}

// Run executes a git command and returns the output
func (g *Git) Run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if g.dir != "" {
		cmd.Dir = g.dir
	}
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if err != nil {
		return output, fmt.Errorf("git %s: %s", strings.Join(args, " "), output)
	}
	return output, nil
}

// RunSilent executes a git command and returns only the error
func (g *Git) RunSilent(args ...string) error {
	_, err := g.Run(args...)
	return err
}

// IsRepo checks if the current directory is a git repository
func (g *Git) IsRepo() bool {
	err := g.RunSilent("rev-parse", "--is-inside-work-tree")
	return err == nil
}

// RepoRoot returns the root directory of the git repository
func (g *Git) RepoRoot() (string, error) {
	return g.Run("rev-parse", "--show-toplevel")
}

// CurrentBranch returns the current branch name
func (g *Git) CurrentBranch() (string, error) {
	return g.Run("rev-parse", "--abbrev-ref", "HEAD")
}

// DefaultBranch tries to detect the default branch
func (g *Git) DefaultBranch() string {
	// Try to detect from remote
	out, err := g.Run("symbolic-ref", "refs/remotes/origin/HEAD", "--short")
	if err == nil {
		parts := strings.SplitN(out, "/", 2)
		if len(parts) == 2 {
			return parts[1]
		}
		return out
	}
	// Fallback: check if main exists
	if err := g.RunSilent("show-ref", "--verify", "--quiet", "refs/heads/main"); err == nil {
		return "main"
	}
	return "master"
}

// BranchExists checks if a branch exists locally
func (g *Git) BranchExists(branch string) bool {
	err := g.RunSilent("show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	return err == nil
}

// RemoteBranchExists checks if a branch exists on the remote
func (g *Git) RemoteBranchExists(branch string) bool {
	err := g.RunSilent("ls-remote", "--exit-code", "--heads", "origin", branch)
	return err == nil
}

// CreateBranch creates a new branch from a base
func (g *Git) CreateBranch(name, base string) error {
	return g.RunSilent("checkout", "-b", name, base)
}

// CreateBranchFromCurrent creates a new branch from the current HEAD
func (g *Git) CreateBranchFromCurrent(name string) error {
	return g.RunSilent("checkout", "-b", name)
}

// Checkout switches to a branch
func (g *Git) Checkout(branch string) error {
	return g.RunSilent("checkout", branch)
}

// Fetch fetches from remote
func (g *Git) Fetch() error {
	return g.RunSilent("fetch", "--prune")
}

// FetchBranch fetches a specific branch from remote
func (g *Git) FetchBranch(branch string) error {
	return g.RunSilent("fetch", "origin", branch)
}

// Pull pulls the current branch from remote
func (g *Git) Pull() error {
	return g.RunSilent("pull")
}

// PullRebase pulls with rebase
func (g *Git) PullRebase() error {
	return g.RunSilent("pull", "--rebase")
}

// Push pushes the current branch to remote
func (g *Git) Push() error {
	branch, err := g.CurrentBranch()
	if err != nil {
		return err
	}
	return g.RunSilent("push", "origin", branch)
}

// PushSetUpstream pushes and sets upstream
func (g *Git) PushSetUpstream() error {
	branch, err := g.CurrentBranch()
	if err != nil {
		return err
	}
	return g.RunSilent("push", "--set-upstream", "origin", branch)
}

// ForcePushLease pushes with force-with-lease
func (g *Git) ForcePushLease() error {
	branch, err := g.CurrentBranch()
	if err != nil {
		return err
	}
	return g.RunSilent("push", "--force-with-lease", "origin", branch)
}

// Add stages files
func (g *Git) Add(paths ...string) error {
	args := append([]string{"add"}, paths...)
	return g.RunSilent(args...)
}

// AddAll stages all changes
func (g *Git) AddAll() error {
	return g.RunSilent("add", "-A")
}

// Commit creates a commit with the given message
func (g *Git) Commit(message string) error {
	return g.RunSilent("commit", "-m", message)
}

// CommitAllowEmpty creates a commit even if there are no changes
func (g *Git) CommitAllowEmpty(message string) error {
	return g.RunSilent("commit", "--allow-empty", "-m", message)
}

// HasChanges checks if there are uncommitted changes
func (g *Git) HasChanges() bool {
	out, _ := g.Run("status", "--porcelain")
	return out != ""
}

// HasStagedChanges checks if there are staged changes
func (g *Git) HasStagedChanges() bool {
	err := g.RunSilent("diff", "--cached", "--quiet")
	return err != nil
}

// HasUntrackedFiles checks for untracked files
func (g *Git) HasUntrackedFiles() bool {
	out, _ := g.Run("ls-files", "--others", "--exclude-standard")
	return out != ""
}

// Stash stashes current changes
func (g *Git) Stash(message string) error {
	if message != "" {
		return g.RunSilent("stash", "push", "-m", message)
	}
	return g.RunSilent("stash")
}

// StashPop pops the last stash
func (g *Git) StashPop() error {
	return g.RunSilent("stash", "pop")
}

// Merge merges a branch into the current branch
func (g *Git) Merge(branch string, noFF bool) error {
	if noFF {
		return g.RunSilent("merge", "--no-ff", branch)
	}
	return g.RunSilent("merge", branch)
}

// Rebase rebases the current branch onto another
func (g *Git) Rebase(onto string) error {
	return g.RunSilent("rebase", onto)
}

// RebaseInteractive starts an interactive rebase
func (g *Git) RebaseInteractive(onto string) error {
	return g.RunSilent("rebase", "-i", onto)
}

// DeleteBranch deletes a local branch
func (g *Git) DeleteBranch(branch string) error {
	return g.RunSilent("branch", "-d", branch)
}

// ForceDeleteBranch force deletes a local branch
func (g *Git) ForceDeleteBranch(branch string) error {
	return g.RunSilent("branch", "-D", branch)
}

// DeleteRemoteBranch deletes a remote branch
func (g *Git) DeleteRemoteBranch(branch string) error {
	return g.RunSilent("push", "origin", "--delete", branch)
}

// Tag creates a tag
func (g *Git) Tag(name, message string) error {
	if message != "" {
		return g.RunSilent("tag", "-a", name, "-m", message)
	}
	return g.RunSilent("tag", name)
}

// PushTag pushes a tag to remote
func (g *Git) PushTag(name string) error {
	return g.RunSilent("push", "origin", name)
}

// PushAllTags pushes all tags to remote
func (g *Git) PushAllTags() error {
	return g.RunSilent("push", "origin", "--tags")
}

// Log returns the log for the current branch
func (g *Git) Log(count int, format string) (string, error) {
	if format == "" {
		format = "%h %s"
	}
	return g.Run("log", fmt.Sprintf("-%d", count), fmt.Sprintf("--pretty=format:%s", format))
}

// LogBetween returns commits between two refs
func (g *Git) LogBetween(from, to, format string) (string, error) {
	if format == "" {
		format = "%h %s"
	}
	return g.Run("log", fmt.Sprintf("%s..%s", from, to), fmt.Sprintf("--pretty=format:%s", format))
}

// RemoteURL returns the remote URL
func (g *Git) RemoteURL() (string, error) {
	return g.Run("remote", "get-url", "origin")
}

// Status returns the status output
func (g *Git) Status() (string, error) {
	return g.Run("status", "--short")
}

// DiffStat returns diff statistics
func (g *Git) DiffStat(base string) (string, error) {
	return g.Run("diff", "--stat", base+"...")
}

// CommitCount returns the number of commits ahead of a base branch
func (g *Git) CommitCount(base string) (int, error) {
	out, err := g.Run("rev-list", "--count", base+"..HEAD")
	if err != nil {
		return 0, err
	}
	var count int
	fmt.Sscanf(out, "%d", &count)
	return count, nil
}

// LastCommitMessage returns the last commit message
func (g *Git) LastCommitMessage() (string, error) {
	return g.Run("log", "-1", "--pretty=format:%s")
}

// LastCommitHash returns the last commit hash
func (g *Git) LastCommitHash() (string, error) {
	return g.Run("rev-parse", "--short", "HEAD")
}

// UserName returns the git user name
func (g *Git) UserName() string {
	out, err := g.Run("config", "user.name")
	if err != nil {
		return ""
	}
	return out
}

// UserEmail returns the git user email
func (g *Git) UserEmail() string {
	out, err := g.Run("config", "user.email")
	if err != nil {
		return ""
	}
	return out
}

// SetConfig sets a git config value
func (g *Git) SetConfig(key, value string) error {
	return g.RunSilent("config", key, value)
}

// ParseRemoteURL extracts owner and repo from a remote URL
func ParseRemoteURL(url string) (owner, repo string, err error) {
	url = strings.TrimSuffix(url, ".git")

	// SSH format: git@github.com:owner/repo
	if strings.Contains(url, "git@") {
		parts := strings.SplitN(url, ":", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid SSH remote URL: %s", url)
		}
		segments := strings.Split(parts[1], "/")
		if len(segments) < 2 {
			return "", "", fmt.Errorf("invalid SSH remote URL: %s", url)
		}
		return segments[len(segments)-2], segments[len(segments)-1], nil
	}

	// HTTPS format: https://github.com/owner/repo
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid HTTPS remote URL: %s", url)
	}
	return parts[len(parts)-2], parts[len(parts)-1], nil
}

// DetectProvider detects the git provider from the remote URL
func DetectProvider(url string) string {
	lower := strings.ToLower(url)
	switch {
	case strings.Contains(lower, "github"):
		return "github"
	case strings.Contains(lower, "gitlab"):
		return "gitlab"
	case strings.Contains(lower, "bitbucket"):
		return "bitbucket"
	default:
		return "github"
	}
}
