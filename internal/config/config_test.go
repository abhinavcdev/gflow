package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Version != Version {
		t.Errorf("expected version %s, got %s", Version, cfg.Version)
	}
	if cfg.Provider.Name != "github" {
		t.Errorf("expected provider github, got %s", cfg.Provider.Name)
	}
	if cfg.Provider.TokenEnv != "GITHUB_TOKEN" {
		t.Errorf("expected token env GITHUB_TOKEN, got %s", cfg.Provider.TokenEnv)
	}
	if cfg.Branching.Main != "main" {
		t.Errorf("expected main branch 'main', got %s", cfg.Branching.Main)
	}
	if cfg.Branching.Prefixes.Feature != "feature/" {
		t.Errorf("expected feature prefix 'feature/', got %s", cfg.Branching.Prefixes.Feature)
	}
	if cfg.PR.MergeMethod != "squash" {
		t.Errorf("expected merge method 'squash', got %s", cfg.PR.MergeMethod)
	}
	if cfg.Commit.Convention != "conventional" {
		t.Errorf("expected commit convention 'conventional', got %s", cfg.Commit.Convention)
	}
	if len(cfg.Commit.Types) == 0 {
		t.Error("expected default commit types to be non-empty")
	}
}

func TestGetBaseBranch(t *testing.T) {
	tests := []struct {
		name       string
		useDevelop bool
		develop    string
		main       string
		expected   string
	}{
		{"no develop", false, "develop", "main", "main"},
		{"with develop", true, "develop", "main", "develop"},
		{"custom develop", true, "staging", "main", "staging"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Branching.UseDevelop = tt.useDevelop
			cfg.Branching.Develop = tt.develop
			cfg.Branching.Main = tt.main

			result := cfg.GetBaseBranch()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetStrategy(t *testing.T) {
	cfg := DefaultConfig()

	// Built-in strategies
	builtins := []string{"feature", "bugfix", "hotfix", "release"}
	for _, name := range builtins {
		s, ok := cfg.GetStrategy(name)
		if !ok {
			t.Errorf("expected built-in strategy %s to exist", name)
		}
		if s.Prefix == "" {
			t.Errorf("expected strategy %s to have a prefix", name)
		}
	}

	// Unknown strategy
	_, ok := cfg.GetStrategy("nonexistent")
	if ok {
		t.Error("expected unknown strategy to not exist")
	}

	// Custom strategy
	cfg.Strategies = map[string]Strategy{
		"experiment": {
			Description: "Experimental",
			BaseBranch:  "develop",
			Prefix:      "exp/",
			MergeMethod: "squash",
		},
	}

	s, ok := cfg.GetStrategy("experiment")
	if !ok {
		t.Error("expected custom strategy 'experiment' to exist")
	}
	if s.Prefix != "exp/" {
		t.Errorf("expected prefix 'exp/', got %s", s.Prefix)
	}
	if s.BaseBranch != "develop" {
		t.Errorf("expected base 'develop', got %s", s.BaseBranch)
	}

	// Custom strategy overrides built-in
	cfg.Strategies["feature"] = Strategy{
		Description: "Custom feature",
		BaseBranch:  "staging",
		Prefix:      "f/",
	}
	s, ok = cfg.GetStrategy("feature")
	if !ok {
		t.Error("expected overridden feature strategy to exist")
	}
	if s.Prefix != "f/" {
		t.Errorf("expected custom prefix 'f/', got %s", s.Prefix)
	}
}

func TestGetToken(t *testing.T) {
	cfg := DefaultConfig()

	// Direct token
	cfg.Provider.Token = "direct-token"
	if got := cfg.GetToken(); got != "direct-token" {
		t.Errorf("expected 'direct-token', got %s", got)
	}

	// Token from env var
	cfg.Provider.Token = ""
	cfg.Provider.TokenEnv = "GFLOW_TEST_TOKEN"
	os.Setenv("GFLOW_TEST_TOKEN", "env-token")
	defer os.Unsetenv("GFLOW_TEST_TOKEN")

	if got := cfg.GetToken(); got != "env-token" {
		t.Errorf("expected 'env-token', got %s", got)
	}

	// Fallback for github
	cfg.Provider.TokenEnv = ""
	cfg.Provider.Name = "github"
	os.Setenv("GITHUB_TOKEN", "gh-token")
	defer os.Unsetenv("GITHUB_TOKEN")

	if got := cfg.GetToken(); got != "gh-token" {
		t.Errorf("expected 'gh-token', got %s", got)
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ConfigFileName)

	cfg := DefaultConfig()
	cfg.Provider.Owner = "test-org"
	cfg.Provider.Repo = "test-repo"
	cfg.Branching.Main = "trunk"
	cfg.PR.MergeMethod = "rebase"

	// Save
	if err := Save(cfg, configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Load
	loaded, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.Provider.Owner != "test-org" {
		t.Errorf("expected owner 'test-org', got %s", loaded.Provider.Owner)
	}
	if loaded.Provider.Repo != "test-repo" {
		t.Errorf("expected repo 'test-repo', got %s", loaded.Provider.Repo)
	}
	if loaded.Branching.Main != "trunk" {
		t.Errorf("expected main 'trunk', got %s", loaded.Branching.Main)
	}
	if loaded.PR.MergeMethod != "rebase" {
		t.Errorf("expected merge method 'rebase', got %s", loaded.PR.MergeMethod)
	}
}

func TestLoadOrDefault(t *testing.T) {
	// When not in a git repo or no config, should return defaults
	cfg := LoadOrDefault()
	if cfg == nil {
		t.Fatal("LoadOrDefault should never return nil")
	}
	if cfg.Provider.Name != "github" {
		t.Errorf("expected default provider 'github', got %s", cfg.Provider.Name)
	}
}

func TestSaveContainsHeader(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ConfigFileName)

	cfg := DefaultConfig()
	if err := Save(cfg, configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	content := string(data)
	if len(content) < 10 {
		t.Fatal("config file seems too short")
	}
	// Should start with a comment header
	if content[0] != '#' {
		t.Error("config file should start with a comment header")
	}
}
