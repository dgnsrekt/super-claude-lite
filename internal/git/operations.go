package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"super-claude-lite/internal/config"
)

// CloneRepository clones the SuperClaude repository to a temporary directory at the fixed commit
func CloneRepository(tempDir string) error {
	// Clone the repository
	cloneCmd := exec.Command("git", "clone", config.RepoURL, tempDir)
	if err := cloneCmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Change to the cloned directory and checkout the specific commit
	checkoutCmd := exec.Command("git", "checkout", config.FixedCommit)
	checkoutCmd.Dir = tempDir
	if err := checkoutCmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout commit %s: %w", config.FixedCommit, err)
	}

	return nil
}

// ValidateGitInstalled checks if git is available on the system
func ValidateGitInstalled() error {
	_, err := exec.LookPath("git")
	if err != nil {
		return fmt.Errorf("git is not installed or not in PATH: %w", err)
	}
	return nil
}

// GetTempCloneDir creates a temporary directory for cloning
func GetTempCloneDir() (string, error) {
	tempDir, err := os.MkdirTemp("", "superclaude-clone-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}
	return tempDir, nil
}

// CleanupTempDir removes the temporary clone directory
func CleanupTempDir(tempDir string) error {
	if tempDir == "" {
		return nil
	}
	return os.RemoveAll(tempDir)
}

// GetSourcePaths returns the full paths to source directories in the cloned repo
func GetSourcePaths(repoDir string) (corePath, commandsPath string) {
	corePath = filepath.Join(repoDir, config.CoreSourcePath)
	commandsPath = filepath.Join(repoDir, config.CommandsSourcePath)
	return
}
