package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	ConfigFileName = ".gflow.yml"
	Version        = "0.1.0"
)

// Config is the root configuration for gflow
type Config struct {
	Version    string            `yaml:"version"`
	Provider   ProviderConfig    `yaml:"provider"`
	Branching  BranchingConfig   `yaml:"branching"`
	PR         PRConfig          `yaml:"pr"`
	Commit     CommitConfig      `yaml:"commit"`
	Strategies map[string]Strategy `yaml:"strategies,omitempty"`
	Hooks      HooksConfig       `yaml:"hooks,omitempty"`
}

// ProviderConfig defines the git hosting provider
type ProviderConfig struct {
	Name     string `yaml:"name"`     // github, gitlab, bitbucket
	Host     string `yaml:"host"`     // custom host for self-hosted
	Token    string `yaml:"token"`    // auth token (prefer env var)
	TokenEnv string `yaml:"token_env"` // env var name for token
	Owner    string `yaml:"owner"`    // repo owner/org
	Repo     string `yaml:"repo"`     // repo name
}

// BranchingConfig defines branch naming conventions
type BranchingConfig struct {
	Main       string            `yaml:"main"`
	Develop    string            `yaml:"develop"`
	Prefixes   BranchPrefixes    `yaml:"prefixes"`
	UseDevelop bool              `yaml:"use_develop"`
	Protection BranchProtection  `yaml:"protection,omitempty"`
}

// BranchPrefixes defines the prefixes for different branch types
type BranchPrefixes struct {
	Feature string `yaml:"feature"`
	Bugfix  string `yaml:"bugfix"`
	Hotfix  string `yaml:"hotfix"`
	Release string `yaml:"release"`
	Support string `yaml:"support"`
}

// BranchProtection defines branch protection rules
type BranchProtection struct {
	RequireLinearHistory bool `yaml:"require_linear_history"`
	RequireReviews       bool `yaml:"require_reviews"`
	MinReviewers         int  `yaml:"min_reviewers"`
}

// PRConfig defines pull request conventions
type PRConfig struct {
	Template      string   `yaml:"template"`
	TemplatePath  string   `yaml:"template_path"`
	DefaultBase   string   `yaml:"default_base"`
	AutoAssign    bool     `yaml:"auto_assign"`
	Reviewers     []string `yaml:"reviewers,omitempty"`
	TeamReviewers []string `yaml:"team_reviewers,omitempty"`
	Labels        []string `yaml:"labels,omitempty"`
	Draft         bool     `yaml:"draft"`
	MergeMethod   string   `yaml:"merge_method"` // merge, squash, rebase
	DeleteBranch  bool     `yaml:"delete_branch_on_merge"`
	TitleFormat   string   `yaml:"title_format"`
	BodyFormat    string   `yaml:"body_format"`
}

// CommitConfig defines commit message conventions
type CommitConfig struct {
	Convention    string   `yaml:"convention"` // conventional, angular, custom
	Scopes        []string `yaml:"scopes,omitempty"`
	RequireScope  bool     `yaml:"require_scope"`
	RequireTicket bool     `yaml:"require_ticket"`
	TicketPattern string   `yaml:"ticket_pattern"` // regex for ticket extraction
	Types         []string `yaml:"types,omitempty"`
}

// Strategy defines a custom workflow strategy
type Strategy struct {
	Description string       `yaml:"description"`
	BaseBranch  string       `yaml:"base_branch"`
	Prefix      string       `yaml:"prefix"`
	MergeMethod string       `yaml:"merge_method"`
	Labels      []string     `yaml:"labels,omitempty"`
	Reviewers   []string     `yaml:"reviewers,omitempty"`
	Template    string       `yaml:"template,omitempty"`
	Steps       []StepConfig `yaml:"steps,omitempty"`
	Hooks       StepHooks    `yaml:"hooks,omitempty"`
}

// StepConfig defines a step in a custom strategy
type StepConfig struct {
	Name    string `yaml:"name"`
	Command string `yaml:"command"`
	OnFail  string `yaml:"on_fail"` // abort, continue, prompt
}

// StepHooks defines hooks for strategy lifecycle
type StepHooks struct {
	PreCreate  []string `yaml:"pre_create,omitempty"`
	PostCreate []string `yaml:"post_create,omitempty"`
	PreMerge   []string `yaml:"pre_merge,omitempty"`
	PostMerge  []string `yaml:"post_merge,omitempty"`
}

// HooksConfig defines global hooks
type HooksConfig struct {
	PreCommit  []string `yaml:"pre_commit,omitempty"`
	PrePush    []string `yaml:"pre_push,omitempty"`
	PrePR      []string `yaml:"pre_pr,omitempty"`
	PostPR     []string `yaml:"post_pr,omitempty"`
	PreMerge   []string `yaml:"pre_merge,omitempty"`
	PostMerge  []string `yaml:"post_merge,omitempty"`
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() *Config {
	return &Config{
		Version: Version,
		Provider: ProviderConfig{
			Name:     "github",
			TokenEnv: "GITHUB_TOKEN",
		},
		Branching: BranchingConfig{
			Main:       "main",
			Develop:    "develop",
			UseDevelop: false,
			Prefixes: BranchPrefixes{
				Feature: "feature/",
				Bugfix:  "bugfix/",
				Hotfix:  "hotfix/",
				Release: "release/",
				Support: "support/",
			},
		},
		PR: PRConfig{
			DefaultBase:  "main",
			AutoAssign:   true,
			Draft:        false,
			MergeMethod:  "squash",
			DeleteBranch: true,
			TitleFormat:  "{{.Type}}: {{.Description}}",
			BodyFormat:   "",
		},
		Commit: CommitConfig{
			Convention:    "conventional",
			RequireScope:  false,
			RequireTicket: false,
			TicketPattern: `[A-Z]+-\d+`,
			Types: []string{
				"feat", "fix", "docs", "style", "refactor",
				"perf", "test", "build", "ci", "chore", "revert",
			},
		},
	}
}

// Load reads config from the nearest .gflow.yml up the directory tree
func Load() (*Config, error) {
	path, err := FindConfigFile()
	if err != nil {
		return nil, err
	}
	return LoadFromFile(path)
}

// LoadOrDefault loads config or returns defaults if not found
func LoadOrDefault() *Config {
	cfg, err := Load()
	if err != nil {
		return DefaultConfig()
	}
	return cfg
}

// LoadFromFile reads config from a specific file
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	return cfg, nil
}

// Save writes config to a file
func Save(cfg *Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	header := []byte("# gflow configuration — https://github.com/abhinavcdev/gflow\n# Generated by gflow init\n\n")
	data = append(header, data...)

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", path, err)
	}

	return nil
}

// FindConfigFile searches up the directory tree for .gflow.yml
func FindConfigFile() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	for {
		path := filepath.Join(dir, ConfigFileName)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("no %s found (run 'gflow init' to create one)", ConfigFileName)
}

// GetToken resolves the provider token from config or environment
func (c *Config) GetToken() string {
	if c.Provider.Token != "" {
		return c.Provider.Token
	}
	if c.Provider.TokenEnv != "" {
		return os.Getenv(c.Provider.TokenEnv)
	}
	// Fallback env vars by provider
	switch c.Provider.Name {
	case "github":
		if t := os.Getenv("GITHUB_TOKEN"); t != "" {
			return t
		}
		return os.Getenv("GH_TOKEN")
	case "gitlab":
		if t := os.Getenv("GITLAB_TOKEN"); t != "" {
			return t
		}
		return os.Getenv("GL_TOKEN")
	case "bitbucket":
		if t := os.Getenv("BITBUCKET_TOKEN"); t != "" {
			return t
		}
		return os.Getenv("BB_TOKEN")
	}
	return ""
}

// GetBaseBranch returns the appropriate base branch
func (c *Config) GetBaseBranch() string {
	if c.Branching.UseDevelop {
		return c.Branching.Develop
	}
	return c.Branching.Main
}

// GetStrategy returns a strategy by name, checking custom strategies first
func (c *Config) GetStrategy(name string) (*Strategy, bool) {
	if c.Strategies != nil {
		if s, ok := c.Strategies[name]; ok {
			return &s, true
		}
	}
	// Built-in strategies
	builtins := map[string]Strategy{
		"feature": {
			Description: "Feature development",
			BaseBranch:  c.GetBaseBranch(),
			Prefix:      c.Branching.Prefixes.Feature,
			MergeMethod: c.PR.MergeMethod,
		},
		"bugfix": {
			Description: "Bug fix",
			BaseBranch:  c.GetBaseBranch(),
			Prefix:      c.Branching.Prefixes.Bugfix,
			MergeMethod: c.PR.MergeMethod,
		},
		"hotfix": {
			Description: "Production hotfix",
			BaseBranch:  c.Branching.Main,
			Prefix:      c.Branching.Prefixes.Hotfix,
			MergeMethod: "merge",
		},
		"release": {
			Description: "Release preparation",
			BaseBranch:  c.GetBaseBranch(),
			Prefix:      c.Branching.Prefixes.Release,
			MergeMethod: "merge",
		},
	}
	if s, ok := builtins[name]; ok {
		return &s, true
	}
	return nil, false
}
