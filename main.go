package main

import (
	"fmt"
	"os"
)

var version = "dev" // Set by build flags

func main() {
	if len(os.Args) < 2 {
		showUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "version", "--version", "-v":
		fmt.Printf("deps %s\n", version)
		return
	case "help", "--help", "-h":
		showUsage()
		return
	case "get":
		if len(os.Args) < 3 {
			fmt.Println("Usage: deps get github.com/user/repo[@ref]")
			os.Exit(1)
		}
		handleGet(os.Args[2])
	case "check":
		handleCheck()
	case "install":
		handleInstall()
	case "update":
		var repoURL string
		if len(os.Args) >= 3 {
			repoURL = os.Args[2]
		}
		handleUpdate(repoURL)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		showUsage()
		os.Exit(1)
	}
}

func showUsage() {
	fmt.Printf("deps %s - Language agnostic dependency manager\n\n", version)
	fmt.Println("Usage:")
	fmt.Println("  deps get github.com/user/repo[@ref]   Add a dependency")
	fmt.Println("  deps check                            Check dependency status")
	fmt.Println("  deps install                          Install missing dependencies")
	fmt.Println("  deps update [github.com/user/repo]    Update dependencies")
	fmt.Println("  deps version                          Show version")
	fmt.Println("  deps help                             Show this help")
}

func handleGet(repoSpec string) {
	// Parse GitHub URL and ref
	repoURL, ref, err := parseGitHubSpec(repoSpec)
	if err != nil {
		fmt.Printf("Error parsing spec: %v\n", err)
		os.Exit(1)
	}

	owner, repo, err := parseGitHubURL(repoURL)
	if err != nil {
		fmt.Printf("Error parsing URL: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Fetching %s", repoURL)
	if ref != "" {
		fmt.Printf("@%s", ref)
	}
	fmt.Println("...")

	// Resolve ref to commit SHA
	sha, resolvedRef, err := resolveRef(owner, repo, ref)
	if err != nil {
		fmt.Printf("Error resolving ref: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Resolved to %s@%s\n", resolvedRef, sha[:8])

	// Download and extract
	err = downloadRepo(owner, repo, sha, repoURL)
	if err != nil {
		fmt.Printf("Error downloading repo: %v\n", err)
		os.Exit(1)
	}

	// Load or create lock file
	lockFile := loadLockFile()

	// Add/update dependency
	originalRef := ref
	if originalRef == "" {
		originalRef = resolvedRef
	}

	lockFile.Dependencies[repoURL] = Dependency{
		Ref: originalRef,
		SHA: sha,
	}

	// Save lock file
	err = saveLockFile(lockFile)
	if err != nil {
		fmt.Printf("Error saving lock file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s Added %s@%s (%s)\n", colorize(colorGreen, "✓"), repoURL, resolvedRef, sha[:8])
}

func handleCheck() {
	lockFile := loadLockFile()

	if len(lockFile.Dependencies) == 0 {
		fmt.Println("No dependencies found in .deps.lock")
		return
	}

	fmt.Printf("Checking %d dependencies:\n\n", len(lockFile.Dependencies))

	allGood := true
	for repoURL, dep := range lockFile.Dependencies {
		status, err := checkDependency(repoURL, dep)
		if err != nil {
			fmt.Printf("✗ %s: ERROR - %v\n", repoURL, err)
			allGood = false
			continue
		}

		switch status {
		case "ok":
			fmt.Printf("%s %s@%s (%s)\n", colorize(colorGreen, "✓"), repoURL, dep.Ref, dep.SHA[:8])
		case "missing":
			fmt.Printf("%s %s: MISSING - run 'deps install'\n", colorize(colorRed, "✗"), repoURL)
			allGood = false
		}
	}

	if allGood {
		fmt.Printf("\n%s All dependencies are up to date\n", colorize(colorGreen, "✓"))
	} else {
		fmt.Printf("\n%s Some dependencies need attention\n", colorize(colorRed, "✗"))
	}
}

func handleInstall() {
	lockFile := loadLockFile()

	if len(lockFile.Dependencies) == 0 {
		fmt.Println("No dependencies found in .deps.lock")
		return
	}

	fmt.Printf("Installing %d dependencies:\n\n", len(lockFile.Dependencies))

	for repoURL, dep := range lockFile.Dependencies {
		status, err := checkDependency(repoURL, dep)
		if err != nil {
			fmt.Printf("✗ %s: ERROR - %v\n", repoURL, err)
			continue
		}

		if status == "ok" {
			fmt.Printf("%s %s@%s (%s) - already installed\n", colorize(colorGreen, "✓"), repoURL, dep.Ref, dep.SHA[:8])
			continue
		}

		fmt.Printf("Installing %s@%s (%s)...\n", repoURL, dep.Ref, dep.SHA[:8])

		owner, repo, err := parseGitHubURL(repoURL)
		if err != nil {
			fmt.Printf("%s Error parsing URL %s: %v\n", colorize(colorRed, "✗"), repoURL, err)
			continue
		}

		err = downloadRepo(owner, repo, dep.SHA, repoURL)
		if err != nil {
			fmt.Printf("%s Error downloading %s: %v\n", colorize(colorRed, "✗"), repoURL, err)
			continue
		}

		fmt.Printf("%s Installed %s@%s (%s)\n", colorize(colorGreen, "✓"), repoURL, dep.Ref, dep.SHA[:8])
	}

	fmt.Printf("\n%s Installation complete\n", colorize(colorGreen, "✓"))
}

func handleUpdate(specificRepo string) {
	lockFile := loadLockFile()

	if len(lockFile.Dependencies) == 0 {
		fmt.Println("No dependencies found in .deps.lock")
		return
	}

	updated := false

	if specificRepo != "" {
		// Update specific repo
		dep, exists := lockFile.Dependencies[specificRepo]
		if !exists {
			fmt.Printf("Dependency %s not found in .deps.lock\n", specificRepo)
			os.Exit(1)
		}
		updated = updateDependency(specificRepo, dep, lockFile)
	} else {
		// Update all dependencies
		fmt.Printf("Checking for updates to %d dependencies:\n\n", len(lockFile.Dependencies))
		for repoURL, dep := range lockFile.Dependencies {
			if updateDependency(repoURL, dep, lockFile) {
				updated = true
			}
		}
		if !updated {
			fmt.Printf("\n%s All dependencies are up to date\n", colorize(colorGreen, "✓"))
		}
	}

	// Only save lock file if something was actually updated
	if updated {
		err := saveLockFile(lockFile)
		if err != nil {
			fmt.Printf("Error saving lock file: %v\n", err)
			os.Exit(1)
		}
	}
}
