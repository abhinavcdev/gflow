package git

import (
	"testing"
)

func TestParseRemoteURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "HTTPS GitHub",
			url:       "https://github.com/myorg/myrepo.git",
			wantOwner: "myorg",
			wantRepo:  "myrepo",
		},
		{
			name:      "HTTPS GitHub no .git",
			url:       "https://github.com/myorg/myrepo",
			wantOwner: "myorg",
			wantRepo:  "myrepo",
		},
		{
			name:      "SSH GitHub",
			url:       "git@github.com:myorg/myrepo.git",
			wantOwner: "myorg",
			wantRepo:  "myrepo",
		},
		{
			name:      "SSH GitHub no .git",
			url:       "git@github.com:myorg/myrepo",
			wantOwner: "myorg",
			wantRepo:  "myrepo",
		},
		{
			name:      "HTTPS GitLab",
			url:       "https://gitlab.com/group/project.git",
			wantOwner: "group",
			wantRepo:  "project",
		},
		{
			name:      "SSH GitLab",
			url:       "git@gitlab.com:group/project.git",
			wantOwner: "group",
			wantRepo:  "project",
		},
		{
			name:      "HTTPS Bitbucket",
			url:       "https://bitbucket.org/team/repo.git",
			wantOwner: "team",
			wantRepo:  "repo",
		},
		{
			name:      "Self-hosted HTTPS",
			url:       "https://git.internal.company.com/platform/backend.git",
			wantOwner: "platform",
			wantRepo:  "backend",
		},
		{
			name:      "SSH nested group GitLab",
			url:       "git@gitlab.com:group/subgroup/project.git",
			wantOwner: "subgroup",
			wantRepo:  "project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := ParseRemoteURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRemoteURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if owner != tt.wantOwner {
					t.Errorf("owner = %q, want %q", owner, tt.wantOwner)
				}
				if repo != tt.wantRepo {
					t.Errorf("repo = %q, want %q", repo, tt.wantRepo)
				}
			}
		})
	}
}

func TestDetectProvider(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://github.com/org/repo.git", "github"},
		{"git@github.com:org/repo.git", "github"},
		{"https://gitlab.com/org/repo.git", "gitlab"},
		{"git@gitlab.com:org/repo.git", "gitlab"},
		{"https://bitbucket.org/org/repo.git", "bitbucket"},
		{"git@bitbucket.org:org/repo.git", "bitbucket"},
		{"https://git.mycompany.com/org/repo.git", "github"}, // default fallback
		{"https://GitHub.com/org/repo.git", "github"},         // case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := DetectProvider(tt.url)
			if got != tt.expected {
				t.Errorf("DetectProvider(%q) = %q, want %q", tt.url, got, tt.expected)
			}
		})
	}
}
