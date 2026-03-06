package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func setupMockGitHub(handler http.HandlerFunc) (*GitHub, *httptest.Server) {
	server := httptest.NewServer(handler)
	gh := &GitHub{
		owner:   "testorg",
		repo:    "testrepo",
		token:   "test-token",
		baseURL: server.URL,
		client:  server.Client(),
	}
	return gh, server
}

func TestGitHubName(t *testing.T) {
	gh := &GitHub{}
	if gh.Name() != "github" {
		t.Errorf("expected 'github', got %s", gh.Name())
	}
}

func TestGitHubRepoURL(t *testing.T) {
	gh := &GitHub{
		owner:   "myorg",
		repo:    "myrepo",
		baseURL: "https://api.github.com",
	}
	expected := "https://github.com/myorg/myrepo"
	if got := gh.RepoURL(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestGitHubRepoURLSelfHosted(t *testing.T) {
	gh := &GitHub{
		owner:   "myorg",
		repo:    "myrepo",
		baseURL: "https://git.company.com/api/v3",
	}
	expected := "https://git.company.com/myorg/myrepo"
	if got := gh.RepoURL(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestGitHubCreatePR(t *testing.T) {
	gh, server := setupMockGitHub(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/pulls") {
			// Verify auth header
			auth := r.Header.Get("Authorization")
			if auth != "Bearer test-token" {
				t.Errorf("expected Bearer auth, got %s", auth)
			}

			// Verify request body
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)

			if body["title"] != "feat: Add auth" {
				t.Errorf("expected title 'feat: Add auth', got %v", body["title"])
			}
			if body["head"] != "feature/auth" {
				t.Errorf("expected head 'feature/auth', got %v", body["head"])
			}
			if body["base"] != "main" {
				t.Errorf("expected base 'main', got %v", body["base"])
			}

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"number":   42,
				"title":    "feat: Add auth",
				"body":     "PR body",
				"html_url": "https://github.com/testorg/testrepo/pull/42",
				"state":    "open",
				"draft":    false,
				"head":     map[string]string{"ref": "feature/auth"},
				"base":     map[string]string{"ref": "main"},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	pr, err := gh.CreatePR(PRCreateOptions{
		Title: "feat: Add auth",
		Body:  "PR body",
		Head:  "feature/auth",
		Base:  "main",
		Draft: false,
	})
	if err != nil {
		t.Fatalf("CreatePR failed: %v", err)
	}

	if pr.Number != 42 {
		t.Errorf("expected PR #42, got #%d", pr.Number)
	}
	if pr.Title != "feat: Add auth" {
		t.Errorf("expected title 'feat: Add auth', got %s", pr.Title)
	}
	if pr.URL != "https://github.com/testorg/testrepo/pull/42" {
		t.Errorf("unexpected URL: %s", pr.URL)
	}
	if pr.Head != "feature/auth" {
		t.Errorf("expected head 'feature/auth', got %s", pr.Head)
	}
}

func TestGitHubGetPR(t *testing.T) {
	gh, server := setupMockGitHub(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/pulls/42") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"number":   42,
				"title":    "feat: Something",
				"body":     "body",
				"html_url": "https://github.com/testorg/testrepo/pull/42",
				"state":    "open",
				"draft":    true,
				"head":     map[string]string{"ref": "feature/something"},
				"base":     map[string]string{"ref": "main"},
				"labels":   []map[string]string{{"name": "enhancement"}},
				"requested_reviewers": []map[string]string{{"login": "alice"}},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	pr, err := gh.GetPR(42)
	if err != nil {
		t.Fatalf("GetPR failed: %v", err)
	}

	if pr.Number != 42 {
		t.Errorf("expected PR #42, got #%d", pr.Number)
	}
	if !pr.Draft {
		t.Error("expected draft PR")
	}
	if len(pr.Labels) != 1 || pr.Labels[0] != "enhancement" {
		t.Errorf("unexpected labels: %v", pr.Labels)
	}
	if len(pr.Reviewers) != 1 || pr.Reviewers[0] != "alice" {
		t.Errorf("unexpected reviewers: %v", pr.Reviewers)
	}
}

func TestGitHubListPRs(t *testing.T) {
	gh, server := setupMockGitHub(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/pulls") {
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"number":   1,
					"title":    "PR one",
					"html_url": "https://github.com/testorg/testrepo/pull/1",
					"state":    "open",
					"draft":    false,
					"head":     map[string]string{"ref": "feature/one"},
					"base":     map[string]string{"ref": "main"},
					"labels":   []map[string]string{},
				},
				{
					"number":   2,
					"title":    "PR two",
					"html_url": "https://github.com/testorg/testrepo/pull/2",
					"state":    "open",
					"draft":    true,
					"head":     map[string]string{"ref": "feature/two"},
					"base":     map[string]string{"ref": "main"},
					"labels":   []map[string]string{{"name": "bug"}},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	prs, err := gh.ListPRs()
	if err != nil {
		t.Fatalf("ListPRs failed: %v", err)
	}

	if len(prs) != 2 {
		t.Fatalf("expected 2 PRs, got %d", len(prs))
	}
	if prs[0].Number != 1 {
		t.Errorf("expected first PR #1, got #%d", prs[0].Number)
	}
	if prs[1].Draft != true {
		t.Error("expected second PR to be draft")
	}
}

func TestGitHubMergePR(t *testing.T) {
	mergeCalled := false
	gh, server := setupMockGitHub(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" && strings.Contains(r.URL.Path, "/merge") {
			mergeCalled = true
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)

			if body["merge_method"] != "squash" {
				t.Errorf("expected squash merge, got %v", body["merge_method"])
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"merged": true,
			})
			return
		}
		// GetPR call for branch deletion
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/pulls/42") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"number": 42,
				"head":   map[string]string{"ref": "feature/test"},
				"base":   map[string]string{"ref": "main"},
			})
			return
		}
		// Delete ref
		if r.Method == "DELETE" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	err := gh.MergePR(42, PRMergeOptions{
		Method:       "squash",
		DeleteBranch: true,
	})
	if err != nil {
		t.Fatalf("MergePR failed: %v", err)
	}
	if !mergeCalled {
		t.Error("merge endpoint was not called")
	}
}

func TestGitHubGetUser(t *testing.T) {
	gh, server := setupMockGitHub(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/user" {
			json.NewEncoder(w).Encode(map[string]string{
				"login": "testuser",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	user, err := gh.GetUser()
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if user != "testuser" {
		t.Errorf("expected 'testuser', got %s", user)
	}
}

func TestGitHubAPIError(t *testing.T) {
	gh, server := setupMockGitHub(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Validation Failed",
			"errors": []map[string]string{
				{"message": "A pull request already exists"},
			},
		})
	})
	defer server.Close()

	_, err := gh.CreatePR(PRCreateOptions{
		Title: "test",
		Head:  "feature/test",
		Base:  "main",
	})

	if err == nil {
		t.Fatal("expected error for 422 response")
	}
	if !strings.Contains(err.Error(), "Validation Failed") {
		t.Errorf("expected validation error, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' in error, got: %s", err.Error())
	}
}

func TestNewGitHubSelfHosted(t *testing.T) {
	gh, err := NewGitHub("org", "repo", "token", "git.company.com")
	if err != nil {
		t.Fatalf("NewGitHub failed: %v", err)
	}
	expected := "https://git.company.com/api/v3"
	if gh.baseURL != expected {
		t.Errorf("expected baseURL %q, got %q", expected, gh.baseURL)
	}
}

func TestNewGitHubDefault(t *testing.T) {
	gh, err := NewGitHub("org", "repo", "token", "")
	if err != nil {
		t.Fatalf("NewGitHub failed: %v", err)
	}
	if gh.baseURL != "https://api.github.com" {
		t.Errorf("expected default baseURL, got %q", gh.baseURL)
	}
}
