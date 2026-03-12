package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- parseGitHubURL tests ---

func TestParseGitHubURL_Valid(t *testing.T) {
	tests := []struct {
		input     string
		wantOwner string
		wantRepo  string
	}{
		{"github.com/user/repo", "user", "repo"},
		{"github.com/some-org/my-repo", "some-org", "my-repo"},
		{"github.com/user/repo/", "user", "repo"},
		{"github.com/UPPER/Case", "UPPER", "Case"},
		{"github.com/user123/repo456", "user123", "repo456"},
		{"github.com/a/b", "a", "b"},
		{"github.com/user/repo.with.dots", "user", "repo.with.dots"},
		{"github.com/user/repo-with-dashes", "user", "repo-with-dashes"},
		{"github.com/user/repo_with_underscores", "user", "repo_with_underscores"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			owner, repo, err := parseGitHubURL(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if owner != tt.wantOwner {
				t.Errorf("owner = %q, want %q", owner, tt.wantOwner)
			}
			if repo != tt.wantRepo {
				t.Errorf("repo = %q, want %q", repo, tt.wantRepo)
			}
		})
	}
}

func TestParseGitHubURL_Invalid(t *testing.T) {
	tests := []string{
		"",
		"github.com",
		"github.com/",
		"github.com/user",
		"github.com/user/",
		"gitlab.com/user/repo",
		"https://github.com/user/repo",
		"github.com/user/repo/extra",
		"github.com/user/repo/extra/path",
		"not-a-url",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, _, err := parseGitHubURL(input)
			if err == nil {
				t.Errorf("expected error for input %q, got nil", input)
			}
		})
	}
}

// --- parseGitHubSpec tests ---

func TestParseGitHubSpec_Valid(t *testing.T) {
	tests := []struct {
		input   string
		wantURL string
		wantRef string
	}{
		{"github.com/user/repo", "github.com/user/repo", ""},
		{"github.com/user/repo@main", "github.com/user/repo", "main"},
		{"github.com/user/repo@v1.0.0", "github.com/user/repo", "v1.0.0"},
		{"github.com/user/repo@abc123", "github.com/user/repo", "abc123"},
		{"github.com/user/repo@feature/branch", "github.com/user/repo", "feature/branch"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			url, ref, err := parseGitHubSpec(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if url != tt.wantURL {
				t.Errorf("url = %q, want %q", url, tt.wantURL)
			}
			if ref != tt.wantRef {
				t.Errorf("ref = %q, want %q", ref, tt.wantRef)
			}
		})
	}
}

func TestParseGitHubSpec_TooManyAtSigns(t *testing.T) {
	_, _, err := parseGitHubSpec("github.com/user/repo@v1@extra")
	if err == nil {
		t.Error("expected error for multiple @ signs, got nil")
	}
}

// --- resolveRef tests (SHA detection, no network) ---

func TestResolveRef_FullSHA(t *testing.T) {
	sha := "abcdef1234567890abcdef1234567890abcdef12"
	gotSHA, gotRef, err := resolveRef("owner", "repo", sha)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotSHA != sha {
		t.Errorf("sha = %q, want %q", gotSHA, sha)
	}
	if gotRef != sha {
		t.Errorf("ref = %q, want %q", gotRef, sha)
	}
}

func TestResolveRef_FullSHA_AllDigits(t *testing.T) {
	sha := "0123456789012345678901234567890123456789"
	gotSHA, gotRef, err := resolveRef("owner", "repo", sha)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotSHA != sha {
		t.Errorf("sha = %q, want %q", gotSHA, sha)
	}
	if gotRef != sha {
		t.Errorf("ref = %q, want %q", gotRef, sha)
	}
}

// --- HTTP-based GitHub API function tests ---

// testGitHubServer creates a mock GitHub API server and configures the package
// globals to use it. Returns a cleanup function that must be deferred.
func testGitHubServer(t *testing.T, handler http.Handler) func() {
	t.Helper()
	srv := httptest.NewServer(handler)

	origClient := httpClient
	origBase := githubAPIBaseURL

	httpClient = srv.Client()
	githubAPIBaseURL = srv.URL

	return func() {
		srv.Close()
		httpClient = origClient
		githubAPIBaseURL = origBase
	}
}

func TestGetLatestCommitSHA(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/repos/testowner/testrepo", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(GitHubRepo{DefaultBranch: "main"})
	})

	mux.HandleFunc("/repos/testowner/testrepo/branches/main", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(GitHubBranch{
			Commit: GitHubCommit{SHA: "abc123def456abc123def456abc123def456abc1"},
		})
	})

	cleanup := testGitHubServer(t, mux)
	defer cleanup()

	sha, branch, err := getLatestCommitSHA("testowner", "testrepo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "main" {
		t.Errorf("branch = %q, want %q", branch, "main")
	}
	if sha != "abc123def456abc123def456abc123def456abc1" {
		t.Errorf("sha = %q, want %q", sha, "abc123def456abc123def456abc123def456abc1")
	}
}

func TestGetLatestCommitSHA_RepoNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/testowner/testrepo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})

	cleanup := testGitHubServer(t, mux)
	defer cleanup()

	_, _, err := getLatestCommitSHA("testowner", "testrepo")
	if err == nil {
		t.Error("expected error for 404 response, got nil")
	}
}

func TestGetBranchCommitSHA(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/testowner/testrepo/branches/develop", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(GitHubBranch{
			Commit: GitHubCommit{SHA: "deadbeef12345678deadbeef12345678deadbeef"},
		})
	})

	cleanup := testGitHubServer(t, mux)
	defer cleanup()

	sha, ref, err := getBranchCommitSHA("testowner", "testrepo", "develop")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref != "develop" {
		t.Errorf("ref = %q, want %q", ref, "develop")
	}
	if sha != "deadbeef12345678deadbeef12345678deadbeef" {
		t.Errorf("sha = %q, want %q", sha, "deadbeef12345678deadbeef12345678deadbeef")
	}
}

func TestGetBranchCommitSHA_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/testowner/testrepo/branches/nope", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})

	cleanup := testGitHubServer(t, mux)
	defer cleanup()

	_, _, err := getBranchCommitSHA("testowner", "testrepo", "nope")
	if err == nil {
		t.Error("expected error for missing branch, got nil")
	}
}

func TestGetTagCommitSHA(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/testowner/testrepo/git/refs/tags/v1.0.0", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(GitHubRef{
			Object: struct {
				SHA string `json:"sha"`
			}{SHA: "caffee12345678caffee12345678caffee123456"},
		})
	})

	cleanup := testGitHubServer(t, mux)
	defer cleanup()

	sha, err := getTagCommitSHA("testowner", "testrepo", "v1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sha != "caffee12345678caffee12345678caffee123456" {
		t.Errorf("sha = %q, want %q", sha, "caffee12345678caffee12345678caffee123456")
	}
}

func TestGetTagCommitSHA_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/testowner/testrepo/git/refs/tags/v999", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})

	cleanup := testGitHubServer(t, mux)
	defer cleanup()

	_, err := getTagCommitSHA("testowner", "testrepo", "v999")
	if err == nil {
		t.Error("expected error for missing tag, got nil")
	}
}

func TestResolveRef_EmptyRef_UsesDefaultBranch(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/testowner/testrepo", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(GitHubRepo{DefaultBranch: "main"})
	})
	mux.HandleFunc("/repos/testowner/testrepo/branches/main", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(GitHubBranch{
			Commit: GitHubCommit{SHA: "aabbccdd11223344aabbccdd11223344aabbccdd"},
		})
	})

	cleanup := testGitHubServer(t, mux)
	defer cleanup()

	sha, ref, err := resolveRef("testowner", "testrepo", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref != "main" {
		t.Errorf("ref = %q, want %q", ref, "main")
	}
	if sha != "aabbccdd11223344aabbccdd11223344aabbccdd" {
		t.Errorf("sha = %q, want %q", sha, "aabbccdd11223344aabbccdd11223344aabbccdd")
	}
}

func TestResolveRef_BranchRef(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/testowner/testrepo/branches/develop", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(GitHubBranch{
			Commit: GitHubCommit{SHA: "1111222233334444555566667777888899990000"},
		})
	})

	cleanup := testGitHubServer(t, mux)
	defer cleanup()

	sha, ref, err := resolveRef("testowner", "testrepo", "develop")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref != "develop" {
		t.Errorf("ref = %q, want %q", ref, "develop")
	}
	if sha != "1111222233334444555566667777888899990000" {
		t.Errorf("sha = %q, want %q", sha, "1111222233334444555566667777888899990000")
	}
}

func TestResolveRef_TagFallback(t *testing.T) {
	mux := http.NewServeMux()

	// Branch lookup returns 404
	mux.HandleFunc("/repos/testowner/testrepo/branches/v1.0.0", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})

	// Tag lookup succeeds
	mux.HandleFunc("/repos/testowner/testrepo/git/refs/tags/v1.0.0", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(GitHubRef{
			Object: struct {
				SHA string `json:"sha"`
			}{SHA: "aabbccdd00112233aabbccdd00112233aabbccdd"},
		})
	})

	cleanup := testGitHubServer(t, mux)
	defer cleanup()

	sha, ref, err := resolveRef("testowner", "testrepo", "v1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref != "v1.0.0" {
		t.Errorf("ref = %q, want %q", ref, "v1.0.0")
	}
	if sha != "aabbccdd00112233aabbccdd00112233aabbccdd" {
		t.Errorf("sha = %q, want %q", sha, "aabbccdd00112233aabbccdd00112233aabbccdd")
	}
}

func TestResolveRef_NeitherBranchNorTag(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/repos/testowner/testrepo/branches/nonexistent", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})
	mux.HandleFunc("/repos/testowner/testrepo/git/refs/tags/nonexistent", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})

	cleanup := testGitHubServer(t, mux)
	defer cleanup()

	_, _, err := resolveRef("testowner", "testrepo", "nonexistent")
	if err == nil {
		t.Error("expected error when ref is neither branch nor tag, got nil")
	}
}
