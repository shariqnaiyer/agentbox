// Package worktree mirrors spawn.sh's git-worktree logic in Go: each agent
// gets an isolated worktree at <dir-of-repo>/wt-<name> on its own branch.
package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Path returns the worktree path for an agent: sibling of the repo named
// wt-<name>, exactly as spawn.sh computes it:
//
//	wt="$(dirname "$repo")/wt-$name"
func Path(repo, name string) string {
	// Clean first so a trailing slash matches bash `dirname` semantics
	// (dirname "/a/b/c/" == "/a/b"), which is what spawn.sh uses.
	return filepath.Join(filepath.Dir(filepath.Clean(repo)), "wt-"+name)
}

// IsGitRepo reports whether path is inside a git work tree.
func IsGitRepo(path string) bool {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--is-inside-work-tree")
	return cmd.Run() == nil
}

// TopLevel returns the repo root for a path inside a work tree.
func TopLevel(path string) (string, error) {
	out, err := exec.Command("git", "-C", path, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// Add creates the worktree if it does not already exist, trying to create the
// branch and falling back to checking out an existing branch — the same
// two-step spawn.sh uses:
//
//	git worktree add "$wt" -b "$branch" || git worktree add "$wt" "$branch"
//
// Returns the worktree path.
func Add(repo, name, branch string) (string, error) {
	if !IsGitRepo(repo) {
		return "", fmt.Errorf("%s is not a git repo", repo)
	}
	wt := Path(repo, name)
	if exists(wt) {
		return wt, nil
	}
	if err := exec.Command("git", "-C", repo, "worktree", "add", wt, "-b", branch).Run(); err != nil {
		// branch may already exist; check it out instead of creating it.
		if err2 := exec.Command("git", "-C", repo, "worktree", "add", wt, branch).Run(); err2 != nil {
			return "", fmt.Errorf("git worktree add failed: %v / %v", err, err2)
		}
	}
	return wt, nil
}

// Remove removes a worktree (used by `box kill --worktree`).
func Remove(repo, wt string) error {
	return exec.Command("git", "-C", repo, "worktree", "remove", "--force", wt).Run()
}

func exists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
