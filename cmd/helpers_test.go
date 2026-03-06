package cmd

import (
	"testing"

	"github.com/abhinavcdev/gflow/internal/config"
)

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"my-feature", "my-feature"},
		{"My Feature", "my-feature"},
		{"fix/login bug", "fix/login-bug"},
		{"JIRA-123-fix-login", "jira-123-fix-login"},
		{"hello world!@#$%", "hello-world"},
		{"--leading-dashes--", "leading-dashes"},
		{"multiple---dashes", "multiple-dashes"},
		{"under_score", "under_score"},
		{"dots.are.ok", "dots.are.ok"},
		{"MiXeD CaSe", "mixed-case"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeBranchName(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeBranchName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestHumanize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"add-user-auth", "Add user auth"},
		{"fix_login_bug", "Fix login bug"},
		{"simple", "Simple"},
		{"", ""},
		{"UPPER", "UPPER"},
		{"a", "A"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := humanize(tt.input)
			if got != tt.expected {
				t.Errorf("humanize(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestMapBranchToCommitType(t *testing.T) {
	tests := []struct {
		branchType string
		expected   string
	}{
		{"feature", "feat"},
		{"feat", "feat"},
		{"bugfix", "fix"},
		{"fix", "fix"},
		{"hotfix", "fix"},
		{"hot", "fix"},
		{"release", "release"},
		{"support", "chore"},
		{"unknown", "feat"},
	}

	for _, tt := range tests {
		t.Run(tt.branchType, func(t *testing.T) {
			got := mapBranchToCommitType(tt.branchType)
			if got != tt.expected {
				t.Errorf("mapBranchToCommitType(%q) = %q, want %q", tt.branchType, got, tt.expected)
			}
		})
	}
}

func TestParseBranchName(t *testing.T) {
	// Set up config for testing
	cfg = defaultTestConfig()

	tests := []struct {
		branch       string
		expectedType string
		expectedName string
	}{
		{"feature/user-auth", "feature", "user-auth"},
		{"bugfix/login-fix", "bugfix", "login-fix"},
		{"hotfix/critical", "hotfix", "critical"},
		{"release/1.0.0", "release", "1.0.0"},
		{"main", "feature", "main"},
		{"feat/quick", "feat", "quick"},
		{"fix/bug", "fix", "bug"},
	}

	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			gotType, gotName := parseBranchName(tt.branch)
			if gotType != tt.expectedType {
				t.Errorf("parseBranchName(%q) type = %q, want %q", tt.branch, gotType, tt.expectedType)
			}
			if gotName != tt.expectedName {
				t.Errorf("parseBranchName(%q) name = %q, want %q", tt.branch, gotName, tt.expectedName)
			}
		})
	}
}

func TestDetectLabels(t *testing.T) {
	cfg = defaultTestConfig()

	tests := []struct {
		branch string
		expect int
	}{
		{"feature/thing", 1}, // enhancement
		{"bugfix/thing", 1},  // bug
		{"hotfix/thing", 2},  // hotfix, priority
		{"release/1.0", 0},   // none
	}

	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			labels := detectLabels(tt.branch)
			if len(labels) != tt.expect {
				t.Errorf("detectLabels(%q) returned %d labels, want %d: %v", tt.branch, len(labels), tt.expect, labels)
			}
		})
	}
}

func TestFormatList(t *testing.T) {
	if got := formatList(nil); got != "(none)" {
		t.Errorf("formatList(nil) = %q, want '(none)'", got)
	}
	if got := formatList([]string{}); got != "(none)" {
		t.Errorf("formatList([]) = %q, want '(none)'", got)
	}
	if got := formatList([]string{"a", "b"}); got != "a, b" {
		t.Errorf("formatList([a,b]) = %q, want 'a, b'", got)
	}
}

func TestBumpVersion(t *testing.T) {
	tests := []struct {
		current  string
		bump     string
		expected string
	}{
		{"1.2.3", "patch", "1.2.4"},
		{"1.2.3", "minor", "1.3.0"},
		{"1.2.3", "major", "2.0.0"},
		{"0.0.1", "patch", "0.0.2"},
		{"0.1.0", "minor", "0.2.0"},
		{"invalid", "patch", "0.1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.current+"/"+tt.bump, func(t *testing.T) {
			got := bumpVersion(tt.current, tt.bump)
			if got != tt.expected {
				t.Errorf("bumpVersion(%q, %q) = %q, want %q", tt.current, tt.bump, got, tt.expected)
			}
		})
	}
}

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		str     string
		pattern string
		match   bool
	}{
		{"feature/user-auth", "fua", true},
		{"feature/user-auth", "auth", true},
		{"bugfix/login", "bfl", true},
		{"main", "xyz", false},
		{"feature/thing", "ft", true},
		{"", "", true},
		{"abc", "", true},
		{"", "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.str+"/"+tt.pattern, func(t *testing.T) {
			got := fuzzyMatch(tt.str, tt.pattern)
			if got != tt.match {
				t.Errorf("fuzzyMatch(%q, %q) = %v, want %v", tt.str, tt.pattern, got, tt.match)
			}
		})
	}
}

func TestExtractCommitType(t *testing.T) {
	tests := []struct {
		subject  string
		expected string
	}{
		{"feat: add auth", "feat"},
		{"fix(login): resolve redirect", "fix"},
		{"docs: update readme", "docs"},
		{"refactor: clean up code", "refactor"},
		{"just a message", ""},
		{"", ""},
		{"unknown: something", ""},
	}

	for _, tt := range tests {
		t.Run(tt.subject, func(t *testing.T) {
			got := extractCommitType(tt.subject)
			if got != tt.expected {
				t.Errorf("extractCommitType(%q) = %q, want %q", tt.subject, got, tt.expected)
			}
		})
	}
}

// defaultTestConfig creates a config for testing
func defaultTestConfig() *config.Config {
	c := config.DefaultConfig()
	return c
}
