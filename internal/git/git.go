package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type Git struct {
	workDir string
}

func New(workDir string) *Git {
	return &Git{workDir: workDir}
}

func (g *Git) run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = g.workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

func (g *Git) IsRepo() bool {
	_, err := g.run("rev-parse", "--git-dir")
	return err == nil
}

// CheckIgnore returns the subset of paths that git would ignore (respecting
// .gitignore, .git/info/exclude, and global excludes). Paths must be relative
// to the repo root. A clean exit with no matches is not an error.
func (g *Git) CheckIgnore(paths []string) (map[string]bool, error) {
	ignored := make(map[string]bool)
	if len(paths) == 0 {
		return ignored, nil
	}

	cmd := exec.Command("git", "check-ignore", "--stdin")
	cmd.Dir = g.workDir
	cmd.Stdin = strings.NewReader(strings.Join(paths, "\n") + "\n")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Exit code 1 means "nothing ignored", which is not a failure.
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return ignored, nil
		}
		return nil, fmt.Errorf("git check-ignore: %w: %s", err, stderr.String())
	}

	for _, line := range splitNonEmpty(strings.TrimSpace(stdout.String())) {
		ignored[line] = true
	}
	return ignored, nil
}

func (g *Git) HeadShort() (string, error) {
	return g.run("rev-parse", "--short", "HEAD")
}

// ResolveShort resolves any ref (HEAD, a branch, a sha) to its short hash.
func (g *Git) ResolveShort(ref string) (string, error) {
	return g.run("rev-parse", "--short", ref)
}

// WorkingTreeFiles returns files changed in the working tree relative to HEAD,
// including staged, unstaged, and untracked (non-ignored) files.
func (g *Git) WorkingTreeFiles() ([]string, error) {
	tracked, err := g.run("diff", "--name-only", "HEAD")
	if err != nil {
		return nil, err
	}
	untracked, err := g.run("ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var files []string
	for _, line := range append(splitNonEmpty(tracked), splitNonEmpty(untracked)...) {
		if !seen[line] {
			seen[line] = true
			files = append(files, line)
		}
	}
	return files, nil
}

// StagedFiles returns files staged in the index relative to HEAD.
func (g *Git) StagedFiles() ([]string, error) {
	out, err := g.run("diff", "--name-only", "--cached")
	if err != nil {
		return nil, err
	}
	return splitNonEmpty(out), nil
}

func (g *Git) HeadFull() (string, error) {
	return g.run("rev-parse", "HEAD")
}

// LastCommit returns the short hash of the most recent commit that touched
// path, or "" if the path has no commits yet (new/untracked). This is the
// "last reviewed" anchor for a doc: code committed after it is unreviewed.
func (g *Git) LastCommit(path string) (string, error) {
	return g.run("log", "-1", "--format=%h", "--", path)
}

// IsAncestor reports whether commit a is an ancestor of commit b (a is older).
// Used to pick the newer of a doc's last commit and an ack floor. A clean
// "not an ancestor" (exit 1) is not an error; unknown commits are.
func (g *Git) IsAncestor(a, b string) (bool, error) {
	cmd := exec.Command("git", "merge-base", "--is-ancestor", a, b)
	cmd.Dir = g.workDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("git merge-base --is-ancestor %s %s: %w: %s", a, b, err, stderr.String())
	}
	return true, nil
}

func (g *Git) CommitInfo(hash string) (string, error) {
	return g.run("log", "-1", "--format=%h (%ar)", hash)
}

func (g *Git) CommitDate(hash string) (string, error) {
	return g.run("log", "-1", "--format=%cs", hash)
}

func (g *Git) CommitSubject(hash string) (string, error) {
	return g.run("log", "-1", "--format=%s", hash)
}

func (g *Git) ChangedFilesBetween(fromHash, toHash string, files []string) ([]string, error) {
	args := []string{"diff", "--name-only", fromHash + ".." + toHash}
	if len(files) > 0 {
		args = append(args, "--")
		args = append(args, files...)
	}

	output, err := g.run(args...)
	if err != nil {
		return nil, err
	}

	if output == "" {
		return nil, nil
	}

	return strings.Split(output, "\n"), nil
}

func (g *Git) CommitsBetween(fromHash, toHash string, files []string) ([]string, error) {
	args := []string{"log", "--oneline", fromHash + ".." + toHash}
	if len(files) > 0 {
		args = append(args, "--")
		args = append(args, files...)
	}

	output, err := g.run(args...)
	if err != nil {
		return nil, err
	}

	if output == "" {
		return nil, nil
	}

	return strings.Split(output, "\n"), nil
}

func (g *Git) CommitDetails(fromHash, toHash string, files []string) ([]CommitDetail, error) {
	args := []string{"log", "--format=%H|%h|%s", fromHash + ".." + toHash}
	if len(files) > 0 {
		args = append(args, "--")
		args = append(args, files...)
	}

	output, err := g.run(args...)
	if err != nil {
		return nil, err
	}

	if output == "" {
		return nil, nil
	}

	lines := strings.Split(output, "\n")
	details := make([]CommitDetail, 0, len(lines))

	for _, line := range lines {
		parts := strings.SplitN(line, "|", 3)
		if len(parts) == 3 {
			details = append(details, CommitDetail{
				Hash:    parts[0],
				Short:   parts[1],
				Subject: parts[2],
			})
		}
	}

	return details, nil
}

func (g *Git) Diff(fromHash, toHash string, files []string) (string, error) {
	args := []string{"diff", fromHash + ".." + toHash}
	if len(files) > 0 {
		args = append(args, "--")
		args = append(args, files...)
	}

	return g.run(args...)
}

// ChangedFilesSince returns files (optionally filtered) that differ between
// fromHash and the working tree, or the index when staged is true.
func (g *Git) ChangedFilesSince(fromHash string, staged bool, files []string) ([]string, error) {
	args := []string{"diff", "--name-only"}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, fromHash)
	if len(files) > 0 {
		args = append(args, "--")
		args = append(args, files...)
	}

	output, err := g.run(args...)
	if err != nil {
		return nil, err
	}
	return splitNonEmpty(output), nil
}

// DiffSince diffs fromHash against the working tree, or the index when staged.
func (g *Git) DiffSince(fromHash string, staged bool, files []string) (string, error) {
	args := []string{"diff"}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, fromHash)
	if len(files) > 0 {
		args = append(args, "--")
		args = append(args, files...)
	}

	return g.run(args...)
}

func (g *Git) ShowCommitDiff(hash string, files []string) (string, error) {
	args := []string{"show", "--format=", hash}
	if len(files) > 0 {
		args = append(args, "--")
		args = append(args, files...)
	}

	return g.run(args...)
}

func (g *Git) FilesChangedInCommit(hash string, filterFiles []string) ([]string, error) {
	args := []string{"diff-tree", "--no-commit-id", "--name-only", "-r", hash}

	output, err := g.run(args...)
	if err != nil {
		return nil, err
	}

	if output == "" {
		return nil, nil
	}

	changedFiles := strings.Split(output, "\n")

	if len(filterFiles) == 0 {
		return changedFiles, nil
	}

	filterSet := make(map[string]bool)
	for _, f := range filterFiles {
		filterSet[f] = true
	}

	filtered := make([]string, 0)
	for _, f := range changedFiles {
		if filterSet[f] {
			filtered = append(filtered, f)
		}
	}

	return filtered, nil
}

type CommitDetail struct {
	Hash    string
	Short   string
	Subject string
}

func splitNonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
