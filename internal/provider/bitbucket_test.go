package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func setupMockBitbucket(handler http.HandlerFunc) (*Bitbucket, *httptest.Server) {
	server := httptest.NewServer(handler)
	bb := &Bitbucket{
		owner:   "testteam",
		repo:    "testrepo",
		token:   "bb-test-token",
		baseURL: server.URL,
		client:  server.Client(),
	}
	return bb, server
}

func TestBitbucketName(t *testing.T) {
	bb := &Bitbucket{}
	if bb.Name() != "bitbucket" {
		t.Errorf("expected 'bitbucket', got %s", bb.Name())
	}
}

func TestBitbucketRepoURL(t *testing.T) {
	bb := &Bitbucket{owner: "myteam", repo: "myrepo"}
	expected := "https://bitbucket.org/myteam/myrepo"
	if got := bb.RepoURL(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestBitbucketCreatePR(t *testing.T) {
	bb, server := setupMockBitbucket(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/pullrequests") {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer bb-test-token" {
				t.Errorf("expected Bearer auth, got %s", auth)
			}

			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)

			if body["title"] != "feat: New feature" {
				t.Errorf("expected title 'feat: New feature', got %v", body["title"])
			}

			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":          5,
				"title":       "feat: New feature",
				"description": "PR body",
				"state":       "OPEN",
				"links": map[string]interface{}{
					"html": map[string]string{"href": "https://bitbucket.org/testteam/testrepo/pull-requests/5"},
				},
				"source": map[string]interface{}{
					"branch": map[string]string{"name": "feature/new-feat"},
				},
				"destination": map[string]interface{}{
					"branch": map[string]string{"name": "main"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	pr, err := bb.CreatePR(PRCreateOptions{
		Title: "feat: New feature",
		Body:  "PR body",
		Head:  "feature/new-feat",
		Base:  "main",
	})
	if err != nil {
		t.Fatalf("CreatePR (Bitbucket) failed: %v", err)
	}

	if pr.Number != 5 {
		t.Errorf("expected PR #5, got #%d", pr.Number)
	}
	if pr.Title != "feat: New feature" {
		t.Errorf("expected title 'feat: New feature', got %s", pr.Title)
	}
	if pr.Head != "feature/new-feat" {
		t.Errorf("expected head 'feature/new-feat', got %s", pr.Head)
	}
	if pr.Base != "main" {
		t.Errorf("expected base 'main', got %s", pr.Base)
	}
}

func TestBitbucketGetPR(t *testing.T) {
	bb, server := setupMockBitbucket(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/pullrequests/5") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":          5,
				"title":       "fix: Login bug",
				"description": "Fixes login",
				"state":       "OPEN",
				"links": map[string]interface{}{
					"html": map[string]string{"href": "https://bitbucket.org/testteam/testrepo/pull-requests/5"},
				},
				"source": map[string]interface{}{
					"branch": map[string]string{"name": "bugfix/login"},
				},
				"destination": map[string]interface{}{
					"branch": map[string]string{"name": "main"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	pr, err := bb.GetPR(5)
	if err != nil {
		t.Fatalf("GetPR (Bitbucket) failed: %v", err)
	}

	if pr.Number != 5 {
		t.Errorf("expected PR #5, got #%d", pr.Number)
	}
	if pr.State != "OPEN" {
		t.Errorf("expected state 'OPEN', got %s", pr.State)
	}
}

func TestBitbucketListPRs(t *testing.T) {
	bb, server := setupMockBitbucket(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/pullrequests") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{
						"id":    1,
						"title": "PR one",
						"state": "OPEN",
						"links": map[string]interface{}{
							"html": map[string]string{"href": "https://bitbucket.org/testteam/testrepo/pull-requests/1"},
						},
						"source":      map[string]interface{}{"branch": map[string]string{"name": "feature/one"}},
						"destination": map[string]interface{}{"branch": map[string]string{"name": "main"}},
					},
					{
						"id":    2,
						"title": "PR two",
						"state": "OPEN",
						"links": map[string]interface{}{
							"html": map[string]string{"href": "https://bitbucket.org/testteam/testrepo/pull-requests/2"},
						},
						"source":      map[string]interface{}{"branch": map[string]string{"name": "feature/two"}},
						"destination": map[string]interface{}{"branch": map[string]string{"name": "main"}},
					},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	prs, err := bb.ListPRs()
	if err != nil {
		t.Fatalf("ListPRs (Bitbucket) failed: %v", err)
	}

	if len(prs) != 2 {
		t.Fatalf("expected 2 PRs, got %d", len(prs))
	}
	if prs[0].Head != "feature/one" {
		t.Errorf("expected head 'feature/one', got %s", prs[0].Head)
	}
}

func TestBitbucketMergePR(t *testing.T) {
	mergeCalled := false
	bb, server := setupMockBitbucket(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/merge") {
			mergeCalled = true
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)

			if body["merge_strategy"] != "squash" {
				t.Errorf("expected squash strategy, got %v", body["merge_strategy"])
			}
			if body["close_source_branch"] != true {
				t.Error("expected close_source_branch=true")
			}

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"state": "MERGED"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	err := bb.MergePR(5, PRMergeOptions{
		Method:       "squash",
		DeleteBranch: true,
	})
	if err != nil {
		t.Fatalf("MergePR (Bitbucket) failed: %v", err)
	}
	if !mergeCalled {
		t.Error("merge endpoint was not called")
	}
}

func TestBitbucketMergePRRebase(t *testing.T) {
	bb, server := setupMockBitbucket(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/merge") {
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)

			if body["merge_strategy"] != "fast_forward" {
				t.Errorf("expected fast_forward for rebase, got %v", body["merge_strategy"])
			}

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"state": "MERGED"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	err := bb.MergePR(5, PRMergeOptions{Method: "rebase", DeleteBranch: false})
	if err != nil {
		t.Fatalf("MergePR rebase (Bitbucket) failed: %v", err)
	}
}

func TestBitbucketGetPRForBranch(t *testing.T) {
	bb, server := setupMockBitbucket(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/pullrequests") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"values": []map[string]interface{}{
					{
						"id":    10,
						"title": "My PR",
						"state": "OPEN",
						"links": map[string]interface{}{
							"html": map[string]string{"href": "https://bitbucket.org/testteam/testrepo/pull-requests/10"},
						},
						"source":      map[string]interface{}{"branch": map[string]string{"name": "feature/target"}},
						"destination": map[string]interface{}{"branch": map[string]string{"name": "main"}},
					},
					{
						"id":    11,
						"title": "Other PR",
						"state": "OPEN",
						"links": map[string]interface{}{
							"html": map[string]string{"href": "https://bitbucket.org/testteam/testrepo/pull-requests/11"},
						},
						"source":      map[string]interface{}{"branch": map[string]string{"name": "feature/other"}},
						"destination": map[string]interface{}{"branch": map[string]string{"name": "main"}},
					},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	pr, err := bb.GetPRForBranch("feature/target")
	if err != nil {
		t.Fatalf("GetPRForBranch failed: %v", err)
	}
	if pr.Number != 10 {
		t.Errorf("expected PR #10, got #%d", pr.Number)
	}

	_, err = bb.GetPRForBranch("feature/nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent branch")
	}
}

func TestBitbucketAPIError(t *testing.T) {
	bb, server := setupMockBitbucket(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{"message": "Access denied"},
		})
	})
	defer server.Close()

	_, err := bb.CreatePR(PRCreateOptions{
		Title: "test",
		Head:  "feature/test",
		Base:  "main",
	})

	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if !strings.Contains(err.Error(), "Access denied") {
		t.Errorf("expected 'Access denied' error, got: %s", err.Error())
	}
}

func TestNewBitbucketDefault(t *testing.T) {
	bb, err := NewBitbucket("team", "repo", "token", "")
	if err != nil {
		t.Fatalf("NewBitbucket failed: %v", err)
	}
	if bb.baseURL != "https://api.bitbucket.org/2.0" {
		t.Errorf("expected default baseURL, got %q", bb.baseURL)
	}
}

func TestNewBitbucketSelfHosted(t *testing.T) {
	bb, err := NewBitbucket("team", "repo", "token", "bitbucket.company.com")
	if err != nil {
		t.Fatalf("NewBitbucket failed: %v", err)
	}
	expected := "https://bitbucket.company.com/rest/api/1.0"
	if bb.baseURL != expected {
		t.Errorf("expected baseURL %q, got %q", expected, bb.baseURL)
	}
}

// Verify interface compliance
var _ Provider = (*Bitbucket)(nil)
var _ Provider = (*GitHub)(nil)
var _ Provider = (*GitLab)(nil)
