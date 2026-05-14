package gitserver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func createTempClone(repoPath string) (string, error) {
	workDir, err := os.MkdirTemp("", "sp-git-edit-*")
	if err != nil {
		return "", err
	}
	cmd := exec.Command("git", "clone", repoPath, ".")
	cmd.Dir = workDir
	if out, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(workDir)
		return "", fmt.Errorf("clone failed: %w: %s", err, out)
	}
	return workDir, nil
}

func removeTempDir(dir string) {
	os.RemoveAll(dir)
}

func writeFileWithDirs(path string, content []byte) error {
	os.MkdirAll(filepath.Dir(path), 0755)
	return os.WriteFile(path, content, 0644)
}

func commitAndPush(workDir, filePath, message, authorName string) error {
	run := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = workDir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME="+authorName,
			"GIT_AUTHOR_EMAIL=ui@superplane.com",
			"GIT_COMMITTER_NAME=SuperPlane",
			"GIT_COMMITTER_EMAIL=system@superplane.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git %s: %w: %s", args[0], err, out)
		}
		return nil
	}

	if err := run("add", filePath); err != nil {
		return err
	}
	if err := run("commit", "-m", message); err != nil {
		return err
	}
	return run("push", "origin", "main")
}
