package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

type GitHubRepo struct {
	DefaultBranch string `json:"default_branch"`
}

type GitHubBranch struct {
	Commit GitHubCommit `json:"commit"`
}

type GitHubCommit struct {
	SHA string `json:"sha"`
}

type GitHubRef struct {
	Object struct {
		SHA string `json:"sha"`
	} `json:"object"`
}

func parseGitHubURL(url string) (owner, repo string, err error) {
	// Handle github.com/owner/repo format
	re := regexp.MustCompile(`^github\.com/([^/]+)/([^/]+)/?$`)
	matches := re.FindStringSubmatch(url)
	if len(matches) != 3 {
		return "", "", fmt.Errorf("invalid GitHub URL format")
	}
	return matches[1], matches[2], nil
}

func parseGitHubSpec(spec string) (repoURL, ref string, err error) {
	parts := strings.Split(spec, "@")
	if len(parts) == 1 {
		return parts[0], "", nil
	}
	if len(parts) == 2 {
		return parts[0], parts[1], nil
	}
	return "", "", fmt.Errorf("invalid spec format")
}

func resolveRef(owner, repo, ref string) (sha, resolvedRef string, err error) {
	if ref == "" {
		// Get default branch
		return getLatestCommitSHA(owner, repo)
	}

	// Check if it's already a commit SHA (40 hex characters)
	if matched, _ := regexp.MatchString("^[a-f0-9]{40}$", ref); matched {
		return ref, ref, nil
	}

	// Try as a branch first
	sha, resolvedRef, err = getBranchCommitSHA(owner, repo, ref)
	if err == nil {
		return sha, resolvedRef, nil
	}

	// Try as a tag
	sha, err = getTagCommitSHA(owner, repo, ref)
	if err == nil {
		return sha, ref, nil
	}

	return "", "", fmt.Errorf("could not resolve ref '%s' as branch or tag", ref)
}

func getLatestCommitSHA(owner, repo string) (sha, defaultBranch string, err error) {
	// First get the default branch
	repoURL := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	resp, err := http.Get(repoURL)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	var repoInfo GitHubRepo
	err = json.Unmarshal(body, &repoInfo)
	if err != nil {
		return "", "", err
	}

	// Now get the latest commit from the default branch
	branchURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/branches/%s", owner, repo, repoInfo.DefaultBranch)
	resp, err = http.Get(branchURL)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("GitHub API returned status %d for branch", resp.StatusCode)
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	var branchInfo GitHubBranch
	err = json.Unmarshal(body, &branchInfo)
	if err != nil {
		return "", "", err
	}

	return branchInfo.Commit.SHA, repoInfo.DefaultBranch, nil
}

func getBranchCommitSHA(owner, repo, branch string) (sha, resolvedRef string, err error) {
	branchURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/branches/%s", owner, repo, branch)
	resp, err := http.Get(branchURL)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("branch not found")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	var branchInfo GitHubBranch
	err = json.Unmarshal(body, &branchInfo)
	if err != nil {
		return "", "", err
	}

	return branchInfo.Commit.SHA, branch, nil
}

func getTagCommitSHA(owner, repo, tag string) (string, error) {
	tagURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/refs/tags/%s", owner, repo, tag)
	resp, err := http.Get(tagURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("tag not found")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var refInfo GitHubRef
	err = json.Unmarshal(body, &refInfo)
	if err != nil {
		return "", err
	}

	return refInfo.Object.SHA, nil
}
