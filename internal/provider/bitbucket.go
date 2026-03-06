package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Bitbucket implements the Provider interface for Bitbucket Cloud
type Bitbucket struct {
	owner   string
	repo    string
	token   string
	baseURL string
	client  *http.Client
}

// NewBitbucket creates a new Bitbucket provider
func NewBitbucket(owner, repo, token, host string) (*Bitbucket, error) {
	baseURL := "https://api.bitbucket.org/2.0"
	if host != "" && host != "bitbucket.org" {
		baseURL = fmt.Sprintf("https://%s/rest/api/1.0", host)
	}
	return &Bitbucket{
		owner:   owner,
		repo:    repo,
		token:   token,
		baseURL: baseURL,
		client:  &http.Client{},
	}, nil
}

func (bb *Bitbucket) Name() string { return "bitbucket" }

func (bb *Bitbucket) RepoURL() string {
	return fmt.Sprintf("https://bitbucket.org/%s/%s", bb.owner, bb.repo)
}

func (bb *Bitbucket) do(method, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	reqURL := fmt.Sprintf("%s%s", bb.baseURL, path)
	req, err := http.NewRequest(method, reqURL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+bb.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := bb.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var bbErr struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(respBody, &bbErr)
		errMsg := bbErr.Error.Message
		if errMsg == "" {
			errMsg = string(respBody)
		}
		return fmt.Errorf("Bitbucket API error (%d): %s", resp.StatusCode, errMsg)
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

func (bb *Bitbucket) CreatePR(opts PRCreateOptions) (*PullRequest, error) {
	payload := map[string]interface{}{
		"title":       opts.Title,
		"description": opts.Body,
		"source": map[string]interface{}{
			"branch": map[string]string{"name": opts.Head},
		},
		"destination": map[string]interface{}{
			"branch": map[string]string{"name": opts.Base},
		},
		"close_source_branch": true,
	}

	var bbPR struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
		Desc  string `json:"description"`
		State string `json:"state"`
		Links struct {
			HTML struct {
				Href string `json:"href"`
			} `json:"html"`
		} `json:"links"`
		Source struct {
			Branch struct {
				Name string `json:"name"`
			} `json:"branch"`
		} `json:"source"`
		Dest struct {
			Branch struct {
				Name string `json:"name"`
			} `json:"branch"`
		} `json:"destination"`
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests", bb.owner, bb.repo)
	if err := bb.do("POST", path, payload, &bbPR); err != nil {
		return nil, err
	}

	return &PullRequest{
		Number: bbPR.ID,
		Title:  bbPR.Title,
		Body:   bbPR.Desc,
		URL:    bbPR.Links.HTML.Href,
		State:  bbPR.State,
		Head:   bbPR.Source.Branch.Name,
		Base:   bbPR.Dest.Branch.Name,
	}, nil
}

func (bb *Bitbucket) GetPR(number int) (*PullRequest, error) {
	var bbPR struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
		Desc  string `json:"description"`
		State string `json:"state"`
		Links struct {
			HTML struct {
				Href string `json:"href"`
			} `json:"html"`
		} `json:"links"`
		Source struct {
			Branch struct {
				Name string `json:"name"`
			} `json:"branch"`
		} `json:"source"`
		Dest struct {
			Branch struct {
				Name string `json:"name"`
			} `json:"branch"`
		} `json:"destination"`
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d", bb.owner, bb.repo, number)
	if err := bb.do("GET", path, nil, &bbPR); err != nil {
		return nil, err
	}

	return &PullRequest{
		Number: bbPR.ID,
		Title:  bbPR.Title,
		Body:   bbPR.Desc,
		URL:    bbPR.Links.HTML.Href,
		State:  bbPR.State,
		Head:   bbPR.Source.Branch.Name,
		Base:   bbPR.Dest.Branch.Name,
	}, nil
}

func (bb *Bitbucket) GetPRForBranch(branch string) (*PullRequest, error) {
	var result struct {
		Values []struct {
			ID    int    `json:"id"`
			Title string `json:"title"`
			Desc  string `json:"description"`
			State string `json:"state"`
			Links struct {
				HTML struct {
					Href string `json:"href"`
				} `json:"html"`
			} `json:"links"`
			Source struct {
				Branch struct {
					Name string `json:"name"`
				} `json:"branch"`
			} `json:"source"`
			Dest struct {
				Branch struct {
					Name string `json:"name"`
				} `json:"branch"`
			} `json:"destination"`
		} `json:"values"`
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests?state=OPEN", bb.owner, bb.repo)
	if err := bb.do("GET", path, nil, &result); err != nil {
		return nil, err
	}

	for _, pr := range result.Values {
		if pr.Source.Branch.Name == branch {
			return &PullRequest{
				Number: pr.ID,
				Title:  pr.Title,
				Body:   pr.Desc,
				URL:    pr.Links.HTML.Href,
				State:  pr.State,
				Head:   pr.Source.Branch.Name,
				Base:   pr.Dest.Branch.Name,
			}, nil
		}
	}

	return nil, fmt.Errorf("no open PR found for branch %s", branch)
}

func (bb *Bitbucket) ListPRs() ([]*PullRequest, error) {
	var result struct {
		Values []struct {
			ID    int    `json:"id"`
			Title string `json:"title"`
			State string `json:"state"`
			Links struct {
				HTML struct {
					Href string `json:"href"`
				} `json:"html"`
			} `json:"links"`
			Source struct {
				Branch struct {
					Name string `json:"name"`
				} `json:"branch"`
			} `json:"source"`
			Dest struct {
				Branch struct {
					Name string `json:"name"`
				} `json:"branch"`
			} `json:"destination"`
		} `json:"values"`
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests?state=OPEN&pagelen=30", bb.owner, bb.repo)
	if err := bb.do("GET", path, nil, &result); err != nil {
		return nil, err
	}

	var prs []*PullRequest
	for _, p := range result.Values {
		prs = append(prs, &PullRequest{
			Number: p.ID,
			Title:  p.Title,
			URL:    p.Links.HTML.Href,
			State:  p.State,
			Head:   p.Source.Branch.Name,
			Base:   p.Dest.Branch.Name,
		})
	}
	return prs, nil
}

func (bb *Bitbucket) MergePR(number int, opts PRMergeOptions) error {
	payload := map[string]interface{}{
		"close_source_branch": opts.DeleteBranch,
	}

	switch opts.Method {
	case "squash":
		payload["merge_strategy"] = "squash"
	case "rebase":
		payload["merge_strategy"] = "fast_forward"
	default:
		payload["merge_strategy"] = "merge_commit"
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/merge", bb.owner, bb.repo, number)
	return bb.do("POST", path, payload, nil)
}

func (bb *Bitbucket) ClosePR(number int) error {
	payload := map[string]interface{}{
		"state": "DECLINED",
	}
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d", bb.owner, bb.repo, number)
	return bb.do("PUT", path, payload, nil)
}

func (bb *Bitbucket) AddReviewers(number int, reviewers, teamReviewers []string) error {
	// Bitbucket requires UUIDs for reviewers; simplified for now
	return nil
}

func (bb *Bitbucket) AddLabels(number int, labels []string) error {
	// Bitbucket Cloud doesn't support PR labels natively
	return nil
}

func (bb *Bitbucket) CreateRelease(opts ReleaseCreateOptions) (*Release, error) {
	// Bitbucket doesn't have a native releases feature like GitHub/GitLab
	// We create a tag instead
	return &Release{
		TagName: opts.TagName,
		Name:    opts.Name,
		Body:    opts.Body,
		URL:     fmt.Sprintf("%s/src/%s", bb.RepoURL(), opts.TagName),
	}, nil
}

func (bb *Bitbucket) GetUser() (string, error) {
	var user struct {
		Username string `json:"username"`
	}
	if err := bb.do("GET", "/user", nil, &user); err != nil {
		return "", err
	}
	return user.Username, nil
}

// GetChecks returns CI check statuses for a branch (not yet implemented for Bitbucket)
func (bb *Bitbucket) GetChecks(branch string) ([]CheckStatus, error) {
	return nil, nil
}

// Ensure Bitbucket implements Provider
var _ Provider = (*Bitbucket)(nil)
