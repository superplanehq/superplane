package runcodeagent

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	githubcommon "github.com/superplanehq/superplane/pkg/integrations/github/common"
)

func githubHostFromURL(raw string) githubcommon.Host {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Hostname() == "" {
		return githubcommon.DotCom()
	}
	host, err := githubcommon.ParseHost(parsed.Hostname())
	if err != nil {
		return githubcommon.DotCom()
	}
	return host
}

func githubHostFromRepository(repository string) githubcommon.Host {
	if strings.Contains(repository, "://") {
		return githubHostFromURL(repository)
	}
	return githubcommon.DotCom()
}

func githubHostFromSpec(spec Spec) githubcommon.Host {
	if strings.TrimSpace(spec.PrURL) != "" {
		return githubHostFromURL(spec.PrURL)
	}
	return githubHostFromRepository(spec.Repository)
}

// pullRequestInfo is the subset of a GitHub pull request we need to drive PR mode.
type pullRequestInfo struct {
	Number   int
	State    string
	HTMLURL  string
	HeadRef  string
	HeadRepo string // owner/repo of the head (may be a fork)
	BaseRef  string
	BaseRepo string // owner/repo of the base
}

// isFork reports whether the PR was opened from a fork.
func (p *pullRequestInfo) isFork() bool {
	return p.HeadRepo != "" && p.BaseRepo != "" && !strings.EqualFold(p.HeadRepo, p.BaseRepo)
}

func parsePRURL(prURL string) (owner, repo string, number int, err error) {
	raw := strings.TrimSpace(prURL)
	parsed, parseErr := url.Parse(raw)
	host := githubHostFromURL(raw)
	if parseErr != nil || !host.IsWebHostname(parsed.Hostname()) {
		return "", "", 0, fmt.Errorf("invalid pull request URL %q (expected a GitHub pull request URL)", prURL)
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 4 || parts[0] == "" || parts[1] == "" || parts[2] != "pull" {
		return "", "", 0, fmt.Errorf("invalid pull request URL %q (expected %s/owner/repo/pull/N)", prURL, host.WebURL())
	}
	number, err = strconv.Atoi(parts[3])
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid pull request number in %q", prURL)
	}
	return parts[0], strings.TrimSuffix(parts[1], ".git"), number, nil
}

type githubUserResponse struct {
	Login string `json:"login"`
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// resolveGitHubUser returns the commit author name and email for the user that
// owns the token, so commits can be attributed to them rather than the agent.
func resolveGitHubUser(httpCtx core.HTTPContext, token string, host githubcommon.Host) (name, email string, err error) {
	body, err := githubGet(httpCtx, host.APIURLEnv()+"/user", token)
	if err != nil {
		return "", "", err
	}
	var out githubUserResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return "", "", fmt.Errorf("decode user response: %w", err)
	}
	if out.Login == "" {
		return "", "", fmt.Errorf("GitHub user lookup returned no login")
	}
	name = out.Name
	if name == "" {
		name = out.Login
	}
	email = out.Email
	if email == "" {
		// GitHub's no-reply address always accepts pushes and links to the account.
		email = fmt.Sprintf("%d+%s@users.noreply.%s", out.ID, out.Login, host.WebHostname())
	}
	return name, email, nil
}

// githubGet performs an authenticated GitHub API GET and returns the body.
func githubGet(httpCtx core.HTTPContext, url, token string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build GitHub request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := httpCtx.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GitHub request failed: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read GitHub response: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("GitHub request failed (%d): %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

type githubPullResponse struct {
	Number  int    `json:"number"`
	State   string `json:"state"`
	HTMLURL string `json:"html_url"`
	Head    struct {
		Ref  string `json:"ref"`
		Repo *struct {
			FullName string `json:"full_name"`
		} `json:"repo"`
	} `json:"head"`
	Base struct {
		Ref  string `json:"ref"`
		Repo *struct {
			FullName string `json:"full_name"`
		} `json:"repo"`
	} `json:"base"`
}

// resolvePullRequest fetches a PR via the GitHub REST API using the provided token.
func resolvePullRequest(httpCtx core.HTTPContext, prURL, token string) (*pullRequestInfo, error) {
	owner, repo, number, err := parsePRURL(prURL)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d", githubHostFromURL(prURL).APIURLEnv(), owner, repo, number)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build pull request lookup: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	res, err := httpCtx.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pull request lookup failed: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read pull request response: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("pull request lookup failed (%d): %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var out githubPullResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode pull request response: %w", err)
	}

	info := &pullRequestInfo{
		Number:  out.Number,
		State:   out.State,
		HTMLURL: out.HTMLURL,
		HeadRef: out.Head.Ref,
		BaseRef: out.Base.Ref,
	}
	if out.Head.Repo != nil {
		info.HeadRepo = out.Head.Repo.FullName
	}
	if out.Base.Repo != nil {
		info.BaseRepo = out.Base.Repo.FullName
	}
	if info.HTMLURL == "" {
		info.HTMLURL = strings.TrimSpace(prURL)
	}
	return info, nil
}
