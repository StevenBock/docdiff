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

func (g *Git) HeadShort() (string, error) {
	return g.run("rev-parse", "--short", "HEAD")
}

func (g *Git) HeadFull() (string, error) {
	return g.run("rev-parse", "HEAD")
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
