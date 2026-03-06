package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// GitLab implements the Provider interface for GitLab
type GitLab struct {
	owner   string
	repo    string
	token   string
	baseURL string
	client  *http.Client
}

// NewGitLab creates a new GitLab provider
func NewGitLab(owner, repo, token, host string) (*GitLab, error) {
	baseURL := "https://gitlab.com/api/v4"
	if host != "" && host != "gitlab.com" {
		baseURL = fmt.Sprintf("https://%s/api/v4", host)
	}
	return &GitLab{
		owner:   owner,
		repo:    repo,
		token:   token,
		baseURL: baseURL,
		client:  &http.Client{},
	}, nil
}

func (gl *GitLab) Name() string { return "gitlab" }

func (gl *GitLab) RepoURL() string {
	host := "gitlab.com"
	if !strings.Contains(gl.baseURL, "gitlab.com") {
		host = strings.TrimPrefix(strings.TrimPrefix(gl.baseURL, "https://"), "http://")
		host = strings.TrimSuffix(host, "/api/v4")
	}
	return fmt.Sprintf("https://%s/%s/%s", host, gl.owner, gl.repo)
}

func (gl *GitLab) projectPath() string {
	return url.PathEscape(fmt.Sprintf("%s/%s", gl.owner, gl.repo))
}

func (gl *GitLab) do(method, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	reqURL := fmt.Sprintf("%s%s", gl.baseURL, path)
	req, err := http.NewRequest(method, reqURL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("PRIVATE-TOKEN", gl.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := gl.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var glErr struct {
			Message interface{} `json:"message"`
			Error   string      `json:"error"`
		}
		_ = json.Unmarshal(respBody, &glErr)
		errMsg := fmt.Sprintf("%v", glErr.Message)
		if glErr.Error != "" {
			errMsg = glErr.Error
		}
		return fmt.Errorf("GitLab API error (%d): %s", resp.StatusCode, errMsg)
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

func (gl *GitLab) CreatePR(opts PRCreateOptions) (*PullRequest, error) {
	payload := map[string]interface{}{
		"title":         opts.Title,
		"description":   opts.Body,
		"source_branch": opts.Head,
		"target_branch": opts.Base,
	}

	if len(opts.Labels) > 0 {
		payload["labels"] = strings.Join(opts.Labels, ",")
	}

	var glMR struct {
		IID    int    `json:"iid"`
		Title  string `json:"title"`
		Desc   string `json:"description"`
		WebURL string `json:"web_url"`
		State  string `json:"state"`
		Draft  bool   `json:"draft"`
		Source string `json:"source_branch"`
		Target string `json:"target_branch"`
	}

	path := fmt.Sprintf("/projects/%s/merge_requests", gl.projectPath())
	if err := gl.do("POST", path, payload, &glMR); err != nil {
		return nil, err
	}

	pr := &PullRequest{
		Number: glMR.IID,
		Title:  glMR.Title,
		Body:   glMR.Desc,
		URL:    glMR.WebURL,
		State:  glMR.State,
		Draft:  glMR.Draft,
		Head:   glMR.Source,
		Base:   glMR.Target,
	}

	if len(opts.Reviewers) > 0 {
		_ = gl.AddReviewers(pr.Number, opts.Reviewers, opts.TeamReviewers)
	}

	return pr, nil
}

func (gl *GitLab) GetPR(number int) (*PullRequest, error) {
	var glMR struct {
		IID    int      `json:"iid"`
		Title  string   `json:"title"`
		Desc   string   `json:"description"`
		WebURL string   `json:"web_url"`
		State  string   `json:"state"`
		Source string   `json:"source_branch"`
		Target string   `json:"target_branch"`
		Labels []string `json:"labels"`
	}

	path := fmt.Sprintf("/projects/%s/merge_requests/%d", gl.projectPath(), number)
	if err := gl.do("GET", path, nil, &glMR); err != nil {
		return nil, err
	}

	return &PullRequest{
		Number: glMR.IID,
		Title:  glMR.Title,
		Body:   glMR.Desc,
		URL:    glMR.WebURL,
		State:  glMR.State,
		Head:   glMR.Source,
		Base:   glMR.Target,
		Labels: glMR.Labels,
	}, nil
}

func (gl *GitLab) GetPRForBranch(branch string) (*PullRequest, error) {
	var glMRs []struct {
		IID    int    `json:"iid"`
		Title  string `json:"title"`
		Desc   string `json:"description"`
		WebURL string `json:"web_url"`
		State  string `json:"state"`
		Source string `json:"source_branch"`
		Target string `json:"target_branch"`
	}

	path := fmt.Sprintf("/projects/%s/merge_requests?source_branch=%s&state=opened", gl.projectPath(), url.QueryEscape(branch))
	if err := gl.do("GET", path, nil, &glMRs); err != nil {
		return nil, err
	}

	if len(glMRs) == 0 {
		return nil, fmt.Errorf("no open MR found for branch %s", branch)
	}

	mr := glMRs[0]
	return &PullRequest{
		Number: mr.IID,
		Title:  mr.Title,
		Body:   mr.Desc,
		URL:    mr.WebURL,
		State:  mr.State,
		Head:   mr.Source,
		Base:   mr.Target,
	}, nil
}

func (gl *GitLab) ListPRs() ([]*PullRequest, error) {
	var glMRs []struct {
		IID    int      `json:"iid"`
		Title  string   `json:"title"`
		WebURL string   `json:"web_url"`
		State  string   `json:"state"`
		Source string   `json:"source_branch"`
		Target string   `json:"target_branch"`
		Labels []string `json:"labels"`
	}

	path := fmt.Sprintf("/projects/%s/merge_requests?state=opened&per_page=30", gl.projectPath())
	if err := gl.do("GET", path, nil, &glMRs); err != nil {
		return nil, err
	}

	var prs []*PullRequest
	for _, mr := range glMRs {
		prs = append(prs, &PullRequest{
			Number: mr.IID,
			Title:  mr.Title,
			URL:    mr.WebURL,
			State:  mr.State,
			Head:   mr.Source,
			Base:   mr.Target,
			Labels: mr.Labels,
		})
	}
	return prs, nil
}

func (gl *GitLab) MergePR(number int, opts PRMergeOptions) error {
	payload := map[string]interface{}{}

	switch opts.Method {
	case "squash":
		payload["squash"] = true
	case "rebase":
		// GitLab handles rebase differently; merge with rebase
		payload["merge_when_pipeline_succeeds"] = false
	}

	if opts.DeleteBranch {
		payload["should_remove_source_branch"] = true
	}

	path := fmt.Sprintf("/projects/%s/merge_requests/%d/merge", gl.projectPath(), number)
	return gl.do("PUT", path, payload, nil)
}

func (gl *GitLab) ClosePR(number int) error {
	payload := map[string]interface{}{
		"state_event": "close",
	}
	path := fmt.Sprintf("/projects/%s/merge_requests/%d", gl.projectPath(), number)
	return gl.do("PUT", path, payload, nil)
}

func (gl *GitLab) AddReviewers(number int, reviewers, teamReviewers []string) error {
	// GitLab uses user IDs for reviewers, so we'd need to resolve usernames to IDs
	// For now, this is a simplified implementation
	return nil
}

func (gl *GitLab) AddLabels(number int, labels []string) error {
	payload := map[string]interface{}{
		"labels": strings.Join(labels, ","),
	}
	path := fmt.Sprintf("/projects/%s/merge_requests/%d", gl.projectPath(), number)
	return gl.do("PUT", path, payload, nil)
}

func (gl *GitLab) CreateRelease(opts ReleaseCreateOptions) (*Release, error) {
	payload := map[string]interface{}{
		"tag_name":    opts.TagName,
		"name":        opts.Name,
		"description": opts.Body,
		"ref":         opts.Target,
	}

	var glRelease struct {
		TagName string `json:"tag_name"`
		Name    string `json:"name"`
		Desc    string `json:"description"`
	}

	path := fmt.Sprintf("/projects/%s/releases", gl.projectPath())
	if err := gl.do("POST", path, payload, &glRelease); err != nil {
		return nil, err
	}

	return &Release{
		TagName: glRelease.TagName,
		Name:    glRelease.Name,
		Body:    glRelease.Desc,
		URL:     fmt.Sprintf("%s/-/releases/%s", gl.RepoURL(), glRelease.TagName),
	}, nil
}

func (gl *GitLab) GetUser() (string, error) {
	var user struct {
		Username string `json:"username"`
	}
	if err := gl.do("GET", "/user", nil, &user); err != nil {
		return "", err
	}
	return user.Username, nil
}
