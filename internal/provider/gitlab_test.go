package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func setupMockGitLab(handler http.HandlerFunc) (*GitLab, *httptest.Server) {
	server := httptest.NewServer(handler)
	gl := &GitLab{
		owner:   "testgroup",
		repo:    "testrepo",
		token:   "glpat-test-token",
		baseURL: server.URL,
		client:  server.Client(),
	}
	return gl, server
}

func TestGitLabName(t *testing.T) {
	gl := &GitLab{}
	if gl.Name() != "gitlab" {
		t.Errorf("expected 'gitlab', got %s", gl.Name())
	}
}

func TestGitLabRepoURL(t *testing.T) {
	gl := &GitLab{
		owner:   "mygroup",
		repo:    "myrepo",
		baseURL: "https://gitlab.com/api/v4",
	}
	expected := "https://gitlab.com/mygroup/myrepo"
	if got := gl.RepoURL(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestGitLabRepoURLSelfHosted(t *testing.T) {
	gl := &GitLab{
		owner:   "mygroup",
		repo:    "myrepo",
		baseURL: "https://git.company.com/api/v4",
	}
	expected := "https://git.company.com/mygroup/myrepo"
	if got := gl.RepoURL(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestGitLabProjectPath(t *testing.T) {
	gl := &GitLab{owner: "my-group", repo: "my-repo"}
	expected := "my-group%2Fmy-repo"
	if got := gl.projectPath(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestGitLabCreateMR(t *testing.T) {
	gl, server := setupMockGitLab(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/merge_requests") {
			// Verify auth header
			auth := r.Header.Get("PRIVATE-TOKEN")
			if auth != "glpat-test-token" {
				t.Errorf("expected PRIVATE-TOKEN auth, got %s", auth)
			}

			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)

			if body["title"] != "feat: New feature" {
				t.Errorf("expected title 'feat: New feature', got %v", body["title"])
			}
			if body["source_branch"] != "feature/new-feat" {
				t.Errorf("expected source 'feature/new-feat', got %v", body["source_branch"])
			}
			if body["target_branch"] != "main" {
				t.Errorf("expected target 'main', got %v", body["target_branch"])
			}

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"iid":           10,
				"title":         "feat: New feature",
				"description":   "MR body",
				"web_url":       "https://gitlab.com/testgroup/testrepo/-/merge_requests/10",
				"state":         "opened",
				"draft":         false,
				"source_branch": "feature/new-feat",
				"target_branch": "main",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	pr, err := gl.CreatePR(PRCreateOptions{
		Title: "feat: New feature",
		Body:  "MR body",
		Head:  "feature/new-feat",
		Base:  "main",
		Draft: false,
	})
	if err != nil {
		t.Fatalf("CreatePR (GitLab) failed: %v", err)
	}

	if pr.Number != 10 {
		t.Errorf("expected MR !10, got !%d", pr.Number)
	}
	if pr.Title != "feat: New feature" {
		t.Errorf("expected title 'feat: New feature', got %s", pr.Title)
	}
	if pr.Head != "feature/new-feat" {
		t.Errorf("expected head 'feature/new-feat', got %s", pr.Head)
	}
}

func TestGitLabGetMR(t *testing.T) {
	gl, server := setupMockGitLab(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/merge_requests/10") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"iid":           10,
				"title":         "fix: Bug fix",
				"description":   "Fixes a bug",
				"web_url":       "https://gitlab.com/testgroup/testrepo/-/merge_requests/10",
				"state":         "opened",
				"source_branch": "bugfix/login",
				"target_branch": "main",
				"labels":        []string{"bug", "priority"},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	pr, err := gl.GetPR(10)
	if err != nil {
		t.Fatalf("GetPR (GitLab) failed: %v", err)
	}

	if pr.Number != 10 {
		t.Errorf("expected MR !10, got !%d", pr.Number)
	}
	if len(pr.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d: %v", len(pr.Labels), pr.Labels)
	}
	if pr.Labels[0] != "bug" {
		t.Errorf("expected first label 'bug', got %s", pr.Labels[0])
	}
}

func TestGitLabListMRs(t *testing.T) {
	gl, server := setupMockGitLab(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/merge_requests") {
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"iid":           1,
					"title":         "MR one",
					"web_url":       "https://gitlab.com/testgroup/testrepo/-/merge_requests/1",
					"state":         "opened",
					"source_branch": "feature/one",
					"target_branch": "main",
					"labels":        []string{},
				},
				{
					"iid":           2,
					"title":         "MR two",
					"web_url":       "https://gitlab.com/testgroup/testrepo/-/merge_requests/2",
					"state":         "opened",
					"source_branch": "feature/two",
					"target_branch": "main",
					"labels":        []string{"enhancement"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	prs, err := gl.ListPRs()
	if err != nil {
		t.Fatalf("ListPRs (GitLab) failed: %v", err)
	}

	if len(prs) != 2 {
		t.Fatalf("expected 2 MRs, got %d", len(prs))
	}
	if prs[1].Labels[0] != "enhancement" {
		t.Errorf("expected label 'enhancement', got %s", prs[1].Labels[0])
	}
}

func TestGitLabMergeMR(t *testing.T) {
	mergeCalled := false
	gl, server := setupMockGitLab(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" && strings.Contains(r.URL.Path, "/merge") {
			mergeCalled = true
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)

			if body["squash"] != true {
				t.Errorf("expected squash=true, got %v", body["squash"])
			}
			if body["should_remove_source_branch"] != true {
				t.Errorf("expected should_remove_source_branch=true")
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{"state": "merged"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	err := gl.MergePR(10, PRMergeOptions{
		Method:       "squash",
		DeleteBranch: true,
	})
	if err != nil {
		t.Fatalf("MergePR (GitLab) failed: %v", err)
	}
	if !mergeCalled {
		t.Error("merge endpoint was not called")
	}
}

func TestGitLabGetUser(t *testing.T) {
	gl, server := setupMockGitLab(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/user" {
			json.NewEncoder(w).Encode(map[string]string{
				"username": "gluser",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	user, err := gl.GetUser()
	if err != nil {
		t.Fatalf("GetUser (GitLab) failed: %v", err)
	}
	if user != "gluser" {
		t.Errorf("expected 'gluser', got %s", user)
	}
}

func TestGitLabAPIError(t *testing.T) {
	gl, server := setupMockGitLab(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "insufficient_scope",
		})
	})
	defer server.Close()

	_, err := gl.CreatePR(PRCreateOptions{
		Title: "test",
		Head:  "feature/test",
		Base:  "main",
	})

	if err == nil {
		t.Fatal("expected error for 403 response")
	}
	if !strings.Contains(err.Error(), "insufficient_scope") {
		t.Errorf("expected scope error, got: %s", err.Error())
	}
}

func TestNewGitLabSelfHosted(t *testing.T) {
	gl, err := NewGitLab("group", "repo", "token", "git.company.com")
	if err != nil {
		t.Fatalf("NewGitLab failed: %v", err)
	}
	expected := "https://git.company.com/api/v4"
	if gl.baseURL != expected {
		t.Errorf("expected baseURL %q, got %q", expected, gl.baseURL)
	}
}

func TestNewGitLabDefault(t *testing.T) {
	gl, err := NewGitLab("group", "repo", "token", "")
	if err != nil {
		t.Fatalf("NewGitLab failed: %v", err)
	}
	if gl.baseURL != "https://gitlab.com/api/v4" {
		t.Errorf("expected default baseURL, got %q", gl.baseURL)
	}
}
