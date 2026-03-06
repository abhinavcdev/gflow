package provider

import (
	"fmt"

	"github.com/abhinavcdev/gflow/internal/config"
)

// PullRequest represents a pull/merge request
type PullRequest struct {
	Number    int
	Title     string
	Body      string
	URL       string
	State     string
	Draft     bool
	Head      string
	Base      string
	Labels    []string
	Reviewers []string
}

// PRCreateOptions holds options for creating a PR
type PRCreateOptions struct {
	Title         string
	Body          string
	Head          string
	Base          string
	Draft         bool
	Labels        []string
	Reviewers     []string
	TeamReviewers []string
	MergeMethod   string
}

// PRMergeOptions holds options for merging a PR
type PRMergeOptions struct {
	Method       string // merge, squash, rebase
	DeleteBranch bool
}

// Release represents a release
type Release struct {
	TagName string
	Name    string
	Body    string
	URL     string
	Draft   bool
	Pre     bool
}

// ReleaseCreateOptions holds options for creating a release
type ReleaseCreateOptions struct {
	TagName    string
	Name       string
	Body       string
	Draft      bool
	Prerelease bool
	Target     string
}

// Provider defines the interface for git hosting providers
type Provider interface {
	// Name returns the provider name
	Name() string

	// CreatePR creates a pull/merge request
	CreatePR(opts PRCreateOptions) (*PullRequest, error)

	// GetPR gets a PR by number
	GetPR(number int) (*PullRequest, error)

	// GetPRForBranch gets the PR for a branch
	GetPRForBranch(branch string) (*PullRequest, error)

	// ListPRs lists open PRs
	ListPRs() ([]*PullRequest, error)

	// MergePR merges a PR
	MergePR(number int, opts PRMergeOptions) error

	// ClosePR closes a PR without merging
	ClosePR(number int) error

	// AddReviewers adds reviewers to a PR
	AddReviewers(number int, reviewers, teamReviewers []string) error

	// AddLabels adds labels to a PR
	AddLabels(number int, labels []string) error

	// CreateRelease creates a release
	CreateRelease(opts ReleaseCreateOptions) (*Release, error)

	// GetUser returns the authenticated user's login
	GetUser() (string, error)

	// RepoURL returns the web URL of the repo
	RepoURL() string
}

// New creates a new Provider based on config
func New(cfg *config.Config) (Provider, error) {
	token := cfg.GetToken()
	if token == "" {
		return nil, fmt.Errorf("no authentication token found. Set %s or configure token_env in .gflow.yml", getTokenEnvHint(cfg.Provider.Name))
	}

	switch cfg.Provider.Name {
	case "github":
		return NewGitHub(cfg.Provider.Owner, cfg.Provider.Repo, token, cfg.Provider.Host)
	case "gitlab":
		return NewGitLab(cfg.Provider.Owner, cfg.Provider.Repo, token, cfg.Provider.Host)
	case "bitbucket":
		return NewBitbucket(cfg.Provider.Owner, cfg.Provider.Repo, token, cfg.Provider.Host)
	default:
		return nil, fmt.Errorf("unsupported provider: %s (supported: github, gitlab, bitbucket)", cfg.Provider.Name)
	}
}

func getTokenEnvHint(provider string) string {
	switch provider {
	case "github":
		return "GITHUB_TOKEN"
	case "gitlab":
		return "GITLAB_TOKEN"
	case "bitbucket":
		return "BITBUCKET_TOKEN"
	default:
		return "GITHUB_TOKEN"
	}
}
