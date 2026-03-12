package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// --- getDepPath tests ---

func TestGetDepPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"github.com/user/repo", filepath.Join(".deps", "github.com/user/repo")},
		{"github.com/org/project", filepath.Join(".deps", "github.com/org/project")},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := getDepPath(tt.input)
			if got != tt.want {
				t.Errorf("getDepPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- colorize tests ---

func TestColorize_NoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	got := colorize(colorGreen, "hello")
	if got != "hello" {
		t.Errorf("colorize with NO_COLOR = %q, want %q", got, "hello")
	}
}

func TestColorize_DumbTerminal(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "dumb")
	got := colorize(colorRed, "error")
	if got != "error" {
		t.Errorf("colorize with TERM=dumb = %q, want %q", got, "error")
	}
}

// --- Lock file round-trip tests ---

// withTempDir creates a temporary directory, chdirs into it, and returns a
// cleanup function that restores the original directory and removes the temp dir.
func withTempDir(t *testing.T) func() {
	t.Helper()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir, err := os.MkdirTemp("", "deps-test-*")
	if err != nil {
		t.Fatal(err)
	}

	err = os.Chdir(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatal(err)
	}

	return func() {
		os.Chdir(origDir)
		os.RemoveAll(tmpDir)
	}
}

func TestLoadLockFile_NoFile(t *testing.T) {
	cleanup := withTempDir(t)
	defer cleanup()

	lf := loadLockFile()
	if lf == nil {
		t.Fatal("loadLockFile returned nil")
	}
	if len(lf.Dependencies) != 0 {
		t.Errorf("expected 0 dependencies, got %d", len(lf.Dependencies))
	}
}

func TestSaveAndLoadLockFile_RoundTrip(t *testing.T) {
	cleanup := withTempDir(t)
	defer cleanup()

	original := &LockFile{
		Dependencies: map[string]Dependency{
			"github.com/user/repo": {
				Ref:  "main",
				SHA:  "abc123def456abc123def456abc123def456abc1",
				Hash: "sha256hashvalue1234567890abcdef1234567890abcdef1234567890abcdef12",
			},
			"github.com/org/project": {
				Ref: "v1.0.0",
				SHA: "deadbeef12345678deadbeef12345678deadbeef",
			},
		},
	}

	err := saveLockFile(original)
	if err != nil {
		t.Fatalf("saveLockFile error: %v", err)
	}

	loaded := loadLockFile()
	if len(loaded.Dependencies) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(loaded.Dependencies))
	}

	for repoURL, origDep := range original.Dependencies {
		loadedDep, ok := loaded.Dependencies[repoURL]
		if !ok {
			t.Errorf("missing dependency %q after load", repoURL)
			continue
		}
		if loadedDep.Ref != origDep.Ref {
			t.Errorf("%s: ref = %q, want %q", repoURL, loadedDep.Ref, origDep.Ref)
		}
		if loadedDep.SHA != origDep.SHA {
			t.Errorf("%s: sha = %q, want %q", repoURL, loadedDep.SHA, origDep.SHA)
		}
		if loadedDep.Hash != origDep.Hash {
			t.Errorf("%s: hash = %q, want %q", repoURL, loadedDep.Hash, origDep.Hash)
		}
	}
}

func TestLockFile_HashOmitEmpty(t *testing.T) {
	cleanup := withTempDir(t)
	defer cleanup()

	lf := &LockFile{
		Dependencies: map[string]Dependency{
			"github.com/user/repo": {
				Ref: "main",
				SHA: "abc123def456abc123def456abc123def456abc1",
				// Hash intentionally empty
			},
		},
	}

	err := saveLockFile(lf)
	if err != nil {
		t.Fatalf("saveLockFile error: %v", err)
	}

	data, err := os.ReadFile(".deps.lock")
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	// The JSON should not contain "hash" when it's empty
	if bytes.Contains(data, []byte(`"hash"`)) {
		t.Errorf("expected no 'hash' key in JSON when empty, got:\n%s", string(data))
	}
}

func TestLoadLockFile_BackwardCompatNoHash(t *testing.T) {
	cleanup := withTempDir(t)
	defer cleanup()

	// Write an old-format lock file without hash fields
	oldFormat := `{
  "dependencies": {
    "github.com/user/repo": {
      "ref": "main",
      "sha": "abc123def456abc123def456abc123def456abc1"
    }
  }
}`
	err := os.WriteFile(".deps.lock", []byte(oldFormat), 0644)
	if err != nil {
		t.Fatal(err)
	}

	lf := loadLockFile()
	dep, ok := lf.Dependencies["github.com/user/repo"]
	if !ok {
		t.Fatal("dependency not found")
	}
	if dep.Ref != "main" {
		t.Errorf("ref = %q, want %q", dep.Ref, "main")
	}
	if dep.SHA != "abc123def456abc123def456abc123def456abc1" {
		t.Errorf("sha = %q", dep.SHA)
	}
	if dep.Hash != "" {
		t.Errorf("hash should be empty for old format, got %q", dep.Hash)
	}
}

func TestLoadLockFile_InvalidJSON(t *testing.T) {
	cleanup := withTempDir(t)
	defer cleanup()

	err := os.WriteFile(".deps.lock", []byte("not json at all"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	lf := loadLockFile()
	if lf == nil {
		t.Fatal("loadLockFile returned nil on invalid JSON")
	}
	if len(lf.Dependencies) != 0 {
		t.Errorf("expected 0 dependencies on invalid JSON, got %d", len(lf.Dependencies))
	}
}

func TestSaveLockFile_JSONFormat(t *testing.T) {
	cleanup := withTempDir(t)
	defer cleanup()

	lf := &LockFile{
		Dependencies: map[string]Dependency{
			"github.com/user/repo": {
				Ref:  "v2.0.0",
				SHA:  "1234567890abcdef1234567890abcdef12345678",
				Hash: "fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321",
			},
		},
	}

	err := saveLockFile(lf)
	if err != nil {
		t.Fatalf("saveLockFile error: %v", err)
	}

	data, err := os.ReadFile(".deps.lock")
	if err != nil {
		t.Fatal(err)
	}

	// Verify it's valid JSON that round-trips
	var parsed LockFile
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("saved lock file is not valid JSON: %v", err)
	}

	dep := parsed.Dependencies["github.com/user/repo"]
	if dep.Hash != "fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321" {
		t.Errorf("hash did not round-trip: %q", dep.Hash)
	}
}

// --- extractTarball tests ---

// makeTarGz creates an in-memory tar.gz archive with the given files.
// The rootPrefix simulates GitHub's tarball format (e.g. "repo-sha1234/").
func makeTarGz(t *testing.T, rootPrefix string, files map[string]string) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Write the root directory entry
	err := tw.WriteHeader(&tar.Header{
		Name:     rootPrefix,
		Typeflag: tar.TypeDir,
		Mode:     0755,
	})
	if err != nil {
		t.Fatal(err)
	}

	for name, content := range files {
		fullName := rootPrefix + name

		// Create parent directories if needed
		dir := filepath.Dir(name)
		if dir != "." {
			err := tw.WriteHeader(&tar.Header{
				Name:     rootPrefix + dir + "/",
				Typeflag: tar.TypeDir,
				Mode:     0755,
			})
			if err != nil {
				t.Fatal(err)
			}
		}

		err := tw.WriteHeader(&tar.Header{
			Name:     fullName,
			Typeflag: tar.TypeReg,
			Mode:     0644,
			Size:     int64(len(content)),
		})
		if err != nil {
			t.Fatal(err)
		}

		_, err = tw.Write([]byte(content))
		if err != nil {
			t.Fatal(err)
		}
	}

	tw.Close()
	gw.Close()
	return &buf
}

func TestExtractTarball_BasicFiles(t *testing.T) {
	cleanup := withTempDir(t)
	defer cleanup()

	files := map[string]string{
		"README.md":    "# Hello World",
		"main.go":      "package main",
		"lib/utils.go": "package lib",
	}

	tarball := makeTarGz(t, "myrepo-abc1234/", files)

	destPath := filepath.Join("extracted", "repo")
	err := extractTarball(tarball, destPath)
	if err != nil {
		t.Fatalf("extractTarball error: %v", err)
	}

	for name, wantContent := range files {
		path := filepath.Join(destPath, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("file %q not found: %v", name, err)
			continue
		}
		if string(data) != wantContent {
			t.Errorf("file %q content = %q, want %q", name, string(data), wantContent)
		}
	}
}

func TestExtractTarball_OverwritesExistingDir(t *testing.T) {
	cleanup := withTempDir(t)
	defer cleanup()

	destPath := filepath.Join("extracted", "repo")

	// Create an existing file that should be removed
	os.MkdirAll(destPath, 0755)
	os.WriteFile(filepath.Join(destPath, "old-file.txt"), []byte("old"), 0644)

	tarball := makeTarGz(t, "repo-sha123/", map[string]string{
		"new-file.txt": "new content",
	})

	err := extractTarball(tarball, destPath)
	if err != nil {
		t.Fatalf("extractTarball error: %v", err)
	}

	// Old file should be gone
	if _, err := os.Stat(filepath.Join(destPath, "old-file.txt")); !os.IsNotExist(err) {
		t.Error("old-file.txt should have been removed")
	}

	// New file should exist
	data, err := os.ReadFile(filepath.Join(destPath, "new-file.txt"))
	if err != nil {
		t.Fatal("new-file.txt not found")
	}
	if string(data) != "new content" {
		t.Errorf("content = %q, want %q", string(data), "new content")
	}
}

func TestExtractTarball_EmptyArchive(t *testing.T) {
	cleanup := withTempDir(t)
	defer cleanup()

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	tw.Close()
	gw.Close()

	destPath := "empty-dest"
	err := extractTarball(&buf, destPath)
	if err != nil {
		t.Fatalf("extractTarball error on empty archive: %v", err)
	}

	// Directory should exist but be empty
	entries, err := os.ReadDir(destPath)
	if err != nil {
		t.Fatalf("ReadDir error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty directory, got %d entries", len(entries))
	}
}

func TestExtractTarball_NestedDirectories(t *testing.T) {
	cleanup := withTempDir(t)
	defer cleanup()

	files := map[string]string{
		"a/b/c/deep.txt": "deeply nested",
	}

	tarball := makeTarGz(t, "project-def567/", files)

	destPath := "nested-test"
	err := extractTarball(tarball, destPath)
	if err != nil {
		t.Fatalf("extractTarball error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(destPath, "a", "b", "c", "deep.txt"))
	if err != nil {
		t.Fatalf("deep file not found: %v", err)
	}
	if string(data) != "deeply nested" {
		t.Errorf("content = %q, want %q", string(data), "deeply nested")
	}
}

// --- downloadRepo hash computation tests ---

func TestDownloadRepo_ComputesHash(t *testing.T) {
	cleanup := withTempDir(t)
	defer cleanup()

	files := map[string]string{
		"README.md": "# Test Repo",
	}
	tarball := makeTarGz(t, "testrepo-abc1234567/", files)
	tarballBytes := tarball.Bytes()

	// Compute expected hash
	hasher := sha256.New()
	hasher.Write(tarballBytes)
	expectedHash := hex.EncodeToString(hasher.Sum(nil))

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/testowner/testrepo/tarball/abc1234567", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.Write(tarballBytes)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	origClient := httpClient
	origBase := githubAPIBaseURL
	httpClient = srv.Client()
	githubAPIBaseURL = srv.URL
	defer func() {
		httpClient = origClient
		githubAPIBaseURL = origBase
	}()

	hash, err := downloadRepo("testowner", "testrepo", "abc1234567", "github.com/testowner/testrepo")
	if err != nil {
		t.Fatalf("downloadRepo error: %v", err)
	}

	if hash != expectedHash {
		t.Errorf("hash = %q, want %q", hash, expectedHash)
	}

	// Verify files were actually extracted
	data, err := os.ReadFile(filepath.Join(".deps", "github.com", "testowner", "testrepo", "README.md"))
	if err != nil {
		t.Fatalf("extracted file not found: %v", err)
	}
	if string(data) != "# Test Repo" {
		t.Errorf("file content = %q, want %q", string(data), "# Test Repo")
	}
}

func TestDownloadRepo_HTTPError(t *testing.T) {
	cleanup := withTempDir(t)
	defer cleanup()

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/testowner/testrepo/tarball/badsha", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	origClient := httpClient
	origBase := githubAPIBaseURL
	httpClient = srv.Client()
	githubAPIBaseURL = srv.URL
	defer func() {
		httpClient = origClient
		githubAPIBaseURL = origBase
	}()

	_, err := downloadRepo("testowner", "testrepo", "badsha", "github.com/testowner/testrepo")
	if err == nil {
		t.Error("expected error for 404 response, got nil")
	}
}

func TestDownloadRepo_ConsistentHash(t *testing.T) {
	cleanup := withTempDir(t)
	defer cleanup()

	files := map[string]string{
		"file.txt": "consistent content",
	}
	tarball := makeTarGz(t, "repo-sha123/", files)
	tarballBytes := tarball.Bytes()

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/testowner/testrepo/tarball/sha123", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.Write(tarballBytes)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	origClient := httpClient
	origBase := githubAPIBaseURL
	httpClient = srv.Client()
	githubAPIBaseURL = srv.URL
	defer func() {
		httpClient = origClient
		githubAPIBaseURL = origBase
	}()

	hash1, err := downloadRepo("testowner", "testrepo", "sha123", "github.com/testowner/testrepo")
	if err != nil {
		t.Fatalf("first download error: %v", err)
	}

	// Download again (extractTarball removes existing dir)
	hash2, err := downloadRepo("testowner", "testrepo", "sha123", "github.com/testowner/testrepo")
	if err != nil {
		t.Fatalf("second download error: %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("hashes differ between downloads: %q vs %q", hash1, hash2)
	}
}

// --- checkDependency tests ---

func TestCheckDependency_Missing(t *testing.T) {
	cleanup := withTempDir(t)
	defer cleanup()

	// Set up a mock server so the GitHub API calls don't fail
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	origClient := httpClient
	origBase := githubAPIBaseURL
	httpClient = srv.Client()
	githubAPIBaseURL = srv.URL
	defer func() {
		httpClient = origClient
		githubAPIBaseURL = origBase
	}()

	dep := Dependency{
		Ref: "main",
		SHA: "abc123def456abc123def456abc123def456abc1",
	}

	result, err := checkDependency("github.com/testowner/testrepo", dep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "missing" {
		t.Errorf("status = %q, want %q", result.Status, "missing")
	}
}

func TestCheckDependency_OK(t *testing.T) {
	cleanup := withTempDir(t)
	defer cleanup()

	repoURL := "github.com/testowner/testrepo"
	depPath := getDepPath(repoURL)
	os.MkdirAll(depPath, 0755)

	sha := "abc123def456abc123def456abc123def456abc1"

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/testowner/testrepo/branches/main", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"commit":{"sha":"%s"}}`, sha)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	origClient := httpClient
	origBase := githubAPIBaseURL
	httpClient = srv.Client()
	githubAPIBaseURL = srv.URL
	defer func() {
		httpClient = origClient
		githubAPIBaseURL = origBase
	}()

	dep := Dependency{Ref: "main", SHA: sha}

	result, err := checkDependency(repoURL, dep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "ok" {
		t.Errorf("status = %q, want %q", result.Status, "ok")
	}
}

func TestCheckDependency_UpdateAvailable(t *testing.T) {
	cleanup := withTempDir(t)
	defer cleanup()

	repoURL := "github.com/testowner/testrepo"
	depPath := getDepPath(repoURL)
	os.MkdirAll(depPath, 0755)

	oldSHA := "abc123def456abc123def456abc123def456abc1"
	newSHA := "def456abc123def456abc123def456abc123def4"

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/testowner/testrepo/branches/main", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"commit":{"sha":"%s"}}`, newSHA)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	origClient := httpClient
	origBase := githubAPIBaseURL
	httpClient = srv.Client()
	githubAPIBaseURL = srv.URL
	defer func() {
		httpClient = origClient
		githubAPIBaseURL = origBase
	}()

	dep := Dependency{Ref: "main", SHA: oldSHA}

	result, err := checkDependency(repoURL, dep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "update_available" {
		t.Errorf("status = %q, want %q", result.Status, "update_available")
	}
	if result.LatestSHA != newSHA {
		t.Errorf("latestSHA = %q, want %q", result.LatestSHA, newSHA)
	}
}

// --- Hash verification integration test ---

func TestHashVerification_TarballIntegrity(t *testing.T) {
	// Verify that the hash computed by TeeReader over the raw tarball bytes
	// matches a manually computed sha256 of the same bytes
	files := map[string]string{
		"hello.txt":     "hello world",
		"sub/nested.go": "package sub",
	}
	tarball := makeTarGz(t, "myrepo-sha999/", files)
	tarballBytes := tarball.Bytes()

	// Manual hash
	h := sha256.New()
	h.Write(tarballBytes)
	expectedHash := hex.EncodeToString(h.Sum(nil))

	// Simulate what downloadRepo does: TeeReader through a hasher
	hasher := sha256.New()
	reader := io.TeeReader(bytes.NewReader(tarballBytes), hasher)

	// Must fully consume the reader (extractTarball does this)
	tmpDir, _ := os.MkdirTemp("", "hash-test-*")
	defer os.RemoveAll(tmpDir)
	destPath := filepath.Join(tmpDir, "output")

	err := extractTarball(reader, destPath)
	if err != nil {
		t.Fatalf("extractTarball error: %v", err)
	}

	gotHash := hex.EncodeToString(hasher.Sum(nil))
	if gotHash != expectedHash {
		t.Errorf("hash = %q, want %q", gotHash, expectedHash)
	}
}
