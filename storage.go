package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type LockFile struct {
	Dependencies map[string]Dependency `json:"dependencies"`
}

type Dependency struct {
	Ref string `json:"ref"`
	SHA string `json:"sha"`
}

func loadLockFile() *LockFile {
	lockFile := &LockFile{
		Dependencies: make(map[string]Dependency),
	}

	data, err := os.ReadFile(".deps.lock")
	if err != nil {
		// File doesn't exist, return empty lock file
		return lockFile
	}

	err = json.Unmarshal(data, lockFile)
	if err != nil {
		fmt.Printf("Warning: could not parse existing .deps.lock: %v\n", err)
		return &LockFile{
			Dependencies: make(map[string]Dependency),
		}
	}

	return lockFile
}

func saveLockFile(lockFile *LockFile) error {
	data, err := json.MarshalIndent(lockFile, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(".deps.lock", data, 0644)
}

func checkDependency(repoURL string, dep Dependency) (string, error) {
	// Check if directory exists
	depPath := getDepPath(repoURL)
	if _, err := os.Stat(depPath); os.IsNotExist(err) {
		return "missing", nil
	}

	return "ok", nil
}

func updateDependency(repoURL string, dep Dependency, lockFile *LockFile) bool {
	owner, repo, err := parseGitHubURL(repoURL)
	if err != nil {
		fmt.Printf("✗ Error parsing URL %s: %v\n", repoURL, err)
		return false
	}

	// Resolve current state of the original ref
	currentSHA, currentRef, err := resolveRef(owner, repo, dep.Ref)
	if err != nil {
		fmt.Printf("✗ Error resolving %s@%s: %v\n", repoURL, dep.Ref, err)
		return false
	}

	if currentSHA == dep.SHA {
		fmt.Printf("✓ %s@%s (%s) - no update available\n", repoURL, dep.Ref, dep.SHA[:8])
		return false
	}

	fmt.Printf("Update available for %s:\n", repoURL)
	fmt.Printf("  Current: %s (%s)\n", dep.SHA[:8], dep.Ref)
	fmt.Printf("  Latest:  %s (%s)\n", currentSHA[:8], currentRef)

	// Download updated version
	err = downloadRepo(owner, repo, currentSHA, repoURL)
	if err != nil {
		fmt.Printf("✗ Error downloading update: %v\n", err)
		return false
	}

	// Update lock file entry
	lockFile.Dependencies[repoURL] = Dependency{
		Ref: dep.Ref,
		SHA: currentSHA,
	}

	fmt.Printf("✓ Updated %s to %s (%s)\n", repoURL, currentRef, currentSHA[:8])
	return true
}

func downloadRepo(owner, repo, sha, repoURL string) error {
	// Create .deps directory if it doesn't exist
	err := os.MkdirAll(".deps", 0755)
	if err != nil {
		return err
	}

	// Download tarball
	tarballURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/tarball/%s", owner, repo, sha)

	resp, err := http.Get(tarballURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	// Extract tarball
	depPath := getDepPath(repoURL)
	err = extractTarball(resp.Body, depPath)
	if err != nil {
		return err
	}

	fmt.Printf("Downloaded to %s\n", depPath)
	return nil
}

func extractTarball(r io.Reader, destPath string) error {
	// Remove existing directory
	os.RemoveAll(destPath)

	// Create destination directory
	err := os.MkdirAll(destPath, 0755)
	if err != nil {
		return err
	}

	// Open gzip reader
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	// Open tar reader
	tr := tar.NewReader(gzr)

	// Track the root directory name (GitHub adds a prefix like "repo-sha/")
	var rootDir string

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Skip pax headers and other non-standard entries
		if header.Typeflag == tar.TypeXGlobalHeader || header.Name == "pax_global_header" {
			continue
		}

		// Detect the root directory from the first real directory entry
		if rootDir == "" && strings.Contains(header.Name, "/") {
			parts := strings.Split(header.Name, "/")
			if len(parts) > 0 && strings.Contains(parts[0], "-") {
				rootDir = parts[0] + "/"
			}
		}

		// Skip the root directory entry itself
		if header.Typeflag == tar.TypeDir && rootDir != "" && header.Name == rootDir[:len(rootDir)-1] {
			continue
		}

		// Skip entries that don't start with our detected root directory
		if rootDir == "" || !strings.HasPrefix(header.Name, rootDir) {
			continue
		}

		// Remove the root directory prefix to flatten the structure
		name := strings.TrimPrefix(header.Name, rootDir)
		if name == "" {
			continue
		}

		target := filepath.Join(destPath, name)

		switch header.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(target, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
		case tar.TypeReg:
			err = os.MkdirAll(filepath.Dir(target), 0755)
			if err != nil {
				return err
			}

			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			_, err = io.Copy(f, tr)
			f.Close()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func getDepPath(repoURL string) string {
	return filepath.Join(".deps", repoURL)
}
