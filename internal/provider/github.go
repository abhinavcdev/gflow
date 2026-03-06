package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// GitHub implements the Provider interface for GitHub
type GitHub struct {
	owner   string
	repo    string
	token   string
	baseURL string
	client  *http.Client
}

// NewGitHub creates a new GitHub provider
func NewGitHub(owner, repo, token, host string) (*GitHub, error) {
	baseURL := "https://api.github.com"
	if host != "" && host != "github.com" {
		baseURL = fmt.Sprintf("https://%s/api/v3", host)
	}
	return &GitHub{
		owner:   owner,
		repo:    repo,
		token:   token,
		baseURL: baseURL,
		client:  &http.Client{},
	}, nil
}

func (g *GitHub) Name() string { return "github" }

func (g *GitHub) RepoURL() string {
	host := "github.com"
	if !strings.Contains(g.baseURL, "api.github.com") {
		host = strings.TrimPrefix(strings.TrimPrefix(g.baseURL, "https://"), "http://")
		host = strings.TrimSuffix(host, "/api/v3")
	}
	return fmt.Sprintf("https://%s/%s/%s", host, g.owner, g.repo)
}

func (g *GitHub) do(method, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := fmt.Sprintf("%s%s", g.baseURL, path)
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+g.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var ghErr struct {
			Message string `json:"message"`
			Errors  []struct {
				Message string `json:"message"`
			} `json:"errors"`
		}
		_ = json.Unmarshal(respBody, &ghErr)
		errMsg := ghErr.Message
		if len(ghErr.Errors) > 0 {
			var msgs []string
			for _, e := range ghErr.Errors {
				msgs = append(msgs, e.Message)
			}
			errMsg = fmt.Sprintf("%s: %s", errMsg, strings.Join(msgs, "; "))
		}
		return fmt.Errorf("GitHub API error (%d): %s", resp.StatusCode, errMsg)
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

func (g *GitHub) CreatePR(opts PRCreateOptions) (*PullRequest, error) {
	payload := map[string]interface{}{
		"title": opts.Title,
		"body":  opts.Body,
		"head":  opts.Head,
		"base":  opts.Base,
		"draft": opts.Draft,
	}

	var ghPR struct {
		Number  int    `json:"number"`
		Title   string `json:"title"`
		Body    string `json:"body"`
		HTMLURL string `json:"html_url"`
		State   string `json:"state"`
		Draft   bool   `json:"draft"`
		Head    struct {
			Ref string `json:"ref"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
	}

	path := fmt.Sprintf("/repos/%s/%s/pulls", g.owner, g.repo)
	if err := g.do("POST", path, payload, &ghPR); err != nil {
		return nil, err
	}

	pr := &PullRequest{
		Number: ghPR.Number,
		Title:  ghPR.Title,
		Body:   ghPR.Body,
		URL:    ghPR.HTMLURL,
		State:  ghPR.State,
		Draft:  ghPR.Draft,
		Head:   ghPR.Head.Ref,
		Base:   ghPR.Base.Ref,
	}

	// Add reviewers
	if len(opts.Reviewers) > 0 || len(opts.TeamReviewers) > 0 {
		_ = g.AddReviewers(pr.Number, opts.Reviewers, opts.TeamReviewers)
	}

	// Add labels
	if len(opts.Labels) > 0 {
		_ = g.AddLabels(pr.Number, opts.Labels)
	}

	return pr, nil
}

func (g *GitHub) GetPR(number int) (*PullRequest, error) {
	var ghPR struct {
		Number  int    `json:"number"`
		Title   string `json:"title"`
		Body    string `json:"body"`
		HTMLURL string `json:"html_url"`
		State   string `json:"state"`
		Draft   bool   `json:"draft"`
		Head    struct {
			Ref string `json:"ref"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
		RequestedReviewers []struct {
			Login string `json:"login"`
		} `json:"requested_reviewers"`
	}

	path := fmt.Sprintf("/repos/%s/%s/pulls/%d", g.owner, g.repo, number)
	if err := g.do("GET", path, nil, &ghPR); err != nil {
		return nil, err
	}

	var labels []string
	for _, l := range ghPR.Labels {
		labels = append(labels, l.Name)
	}
	var reviewers []string
	for _, r := range ghPR.RequestedReviewers {
		reviewers = append(reviewers, r.Login)
	}

	return &PullRequest{
		Number:    ghPR.Number,
		Title:     ghPR.Title,
		Body:      ghPR.Body,
		URL:       ghPR.HTMLURL,
		State:     ghPR.State,
		Draft:     ghPR.Draft,
		Head:      ghPR.Head.Ref,
		Base:      ghPR.Base.Ref,
		Labels:    labels,
		Reviewers: reviewers,
	}, nil
}

func (g *GitHub) GetPRForBranch(branch string) (*PullRequest, error) {
	var ghPRs []struct {
		Number  int    `json:"number"`
		Title   string `json:"title"`
		Body    string `json:"body"`
		HTMLURL string `json:"html_url"`
		State   string `json:"state"`
		Draft   bool   `json:"draft"`
		Head    struct {
			Ref string `json:"ref"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
	}

	path := fmt.Sprintf("/repos/%s/%s/pulls?head=%s:%s&state=open", g.owner, g.repo, g.owner, branch)
	if err := g.do("GET", path, nil, &ghPRs); err != nil {
		return nil, err
	}

	if len(ghPRs) == 0 {
		return nil, fmt.Errorf("no open PR found for branch %s", branch)
	}

	pr := ghPRs[0]
	return &PullRequest{
		Number: pr.Number,
		Title:  pr.Title,
		Body:   pr.Body,
		URL:    pr.HTMLURL,
		State:  pr.State,
		Draft:  pr.Draft,
		Head:   pr.Head.Ref,
		Base:   pr.Base.Ref,
	}, nil
}

func (g *GitHub) ListPRs() ([]*PullRequest, error) {
	var ghPRs []struct {
		Number  int    `json:"number"`
		Title   string `json:"title"`
		HTMLURL string `json:"html_url"`
		State   string `json:"state"`
		Draft   bool   `json:"draft"`
		Head    struct {
			Ref string `json:"ref"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}

	path := fmt.Sprintf("/repos/%s/%s/pulls?state=open&per_page=30", g.owner, g.repo)
	if err := g.do("GET", path, nil, &ghPRs); err != nil {
		return nil, err
	}

	var prs []*PullRequest
	for _, p := range ghPRs {
		var labels []string
		for _, l := range p.Labels {
			labels = append(labels, l.Name)
		}
		prs = append(prs, &PullRequest{
			Number: p.Number,
			Title:  p.Title,
			URL:    p.HTMLURL,
			State:  p.State,
			Draft:  p.Draft,
			Head:   p.Head.Ref,
			Base:   p.Base.Ref,
			Labels: labels,
		})
	}
	return prs, nil
}

func (g *GitHub) MergePR(number int, opts PRMergeOptions) error {
	method := "squash"
	switch opts.Method {
	case "merge":
		method = "merge"
	case "rebase":
		method = "rebase"
	}

	payload := map[string]interface{}{
		"merge_method": method,
	}

	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/merge", g.owner, g.repo, number)
	if err := g.do("PUT", path, payload, nil); err != nil {
		return err
	}

	if opts.DeleteBranch {
		// Get the PR to find the head branch
		pr, err := g.GetPR(number)
		if err == nil {
			deletePath := fmt.Sprintf("/repos/%s/%s/git/refs/heads/%s", g.owner, g.repo, pr.Head)
			_ = g.do("DELETE", deletePath, nil, nil)
		}
	}

	return nil
}

func (g *GitHub) ClosePR(number int) error {
	payload := map[string]interface{}{
		"state": "closed",
	}
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d", g.owner, g.repo, number)
	return g.do("PATCH", path, payload, nil)
}

func (g *GitHub) AddReviewers(number int, reviewers, teamReviewers []string) error {
	payload := map[string]interface{}{}
	if len(reviewers) > 0 {
		payload["reviewers"] = reviewers
	}
	if len(teamReviewers) > 0 {
		payload["team_reviewers"] = teamReviewers
	}
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/requested_reviewers", g.owner, g.repo, number)
	return g.do("POST", path, payload, nil)
}

func (g *GitHub) AddLabels(number int, labels []string) error {
	payload := map[string]interface{}{
		"labels": labels,
	}
	path := fmt.Sprintf("/repos/%s/%s/issues/%d/labels", g.owner, g.repo, number)
	return g.do("POST", path, payload, nil)
}

func (g *GitHub) CreateRelease(opts ReleaseCreateOptions) (*Release, error) {
	payload := map[string]interface{}{
		"tag_name":         opts.TagName,
		"name":             opts.Name,
		"body":             opts.Body,
		"draft":            opts.Draft,
		"prerelease":       opts.Prerelease,
		"target_commitish": opts.Target,
	}

	var ghRelease struct {
		TagName string `json:"tag_name"`
		Name    string `json:"name"`
		Body    string `json:"body"`
		HTMLURL string `json:"html_url"`
		Draft   bool   `json:"draft"`
		Pre     bool   `json:"prerelease"`
	}

	path := fmt.Sprintf("/repos/%s/%s/releases", g.owner, g.repo)
	if err := g.do("POST", path, payload, &ghRelease); err != nil {
		return nil, err
	}

	return &Release{
		TagName: ghRelease.TagName,
		Name:    ghRelease.Name,
		Body:    ghRelease.Body,
		URL:     ghRelease.HTMLURL,
		Draft:   ghRelease.Draft,
		Pre:     ghRelease.Pre,
	}, nil
}

// MarkReady converts a draft PR to ready for review
func (g *GitHub) MarkReady(number int) error {
	payload := map[string]interface{}{
		"draft": false,
	}
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d", g.owner, g.repo, number)
	return g.do("PATCH", path, payload, nil)
}

// ReopenPR reopens a closed PR
func (g *GitHub) ReopenPR(number int) error {
	payload := map[string]interface{}{
		"state": "open",
	}
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d", g.owner, g.repo, number)
	return g.do("PATCH", path, payload, nil)
}

func (g *GitHub) GetUser() (string, error) {
	var user struct {
		Login string `json:"login"`
	}
	if err := g.do("GET", "/user", nil, &user); err != nil {
		return "", err
	}
	return user.Login, nil
}
