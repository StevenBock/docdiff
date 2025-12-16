package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupGitRepo(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@example.com"},
		{"git", "config", "user.name", "Test User"},
	}

	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = tmpDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to run %v: %v", args, err)
		}
	}

	return tmpDir
}

func commitFile(t *testing.T, dir, filename, content, message string) string {
	t.Helper()

	filePath := filepath.Join(dir, filename)
	fileDir := filepath.Dir(filePath)
	os.MkdirAll(fileDir, 0755)

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	cmds := [][]string{
		{"git", "add", filename},
		{"git", "commit", "-m", message},
	}

	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to run %v: %v", args, err)
		}
	}

	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get commit hash: %v", err)
	}

	return strings.TrimSpace(string(out))
}

func TestGit_IsRepo(t *testing.T) {
	t.Run("returns true for git repo", func(t *testing.T) {
		dir := setupGitRepo(t)
		g := New(dir)

		if !g.IsRepo() {
			t.Error("IsRepo() should return true for git repository")
		}
	})

	t.Run("returns false for non-git dir", func(t *testing.T) {
		dir := t.TempDir()
		g := New(dir)

		if g.IsRepo() {
			t.Error("IsRepo() should return false for non-git directory")
		}
	})
}

func TestGit_HeadShort(t *testing.T) {
	dir := setupGitRepo(t)
	commitFile(t, dir, "file.txt", "content", "Initial commit")

	g := New(dir)
	hash, err := g.HeadShort()

	if err != nil {
		t.Fatalf("HeadShort() error = %v", err)
	}

	if len(hash) < 7 {
		t.Errorf("HeadShort() = %s, expected at least 7 chars", hash)
	}
}

func TestGit_HeadFull(t *testing.T) {
	dir := setupGitRepo(t)
	commitFile(t, dir, "file.txt", "content", "Initial commit")

	g := New(dir)
	hash, err := g.HeadFull()

	if err != nil {
		t.Fatalf("HeadFull() error = %v", err)
	}

	if len(hash) != 40 {
		t.Errorf("HeadFull() = %s, expected 40 chars", hash)
	}
}

func TestGit_CommitInfo(t *testing.T) {
	dir := setupGitRepo(t)
	hash := commitFile(t, dir, "file.txt", "content", "Initial commit")

	g := New(dir)
	info, err := g.CommitInfo(hash)

	if err != nil {
		t.Fatalf("CommitInfo() error = %v", err)
	}

	if !strings.Contains(info, hash) {
		t.Errorf("CommitInfo() = %s, should contain hash %s", info, hash)
	}
}

func TestGit_CommitDate(t *testing.T) {
	dir := setupGitRepo(t)
	hash := commitFile(t, dir, "file.txt", "content", "Initial commit")

	g := New(dir)
	date, err := g.CommitDate(hash)

	if err != nil {
		t.Fatalf("CommitDate() error = %v", err)
	}

	if len(date) != 10 {
		t.Errorf("CommitDate() = %s, expected YYYY-MM-DD format", date)
	}
}

func TestGit_CommitSubject(t *testing.T) {
	dir := setupGitRepo(t)
	hash := commitFile(t, dir, "file.txt", "content", "Test commit message")

	g := New(dir)
	subject, err := g.CommitSubject(hash)

	if err != nil {
		t.Fatalf("CommitSubject() error = %v", err)
	}

	if subject != "Test commit message" {
		t.Errorf("CommitSubject() = %s, want 'Test commit message'", subject)
	}
}

func TestGit_ChangedFilesBetween(t *testing.T) {
	dir := setupGitRepo(t)
	hash1 := commitFile(t, dir, "file1.txt", "content1", "First commit")
	commitFile(t, dir, "file2.txt", "content2", "Second commit")
	commitFile(t, dir, "file1.txt", "modified", "Modify first file")

	g := New(dir)

	t.Run("all files", func(t *testing.T) {
		files, err := g.ChangedFilesBetween(hash1, "HEAD", nil)

		if err != nil {
			t.Fatalf("ChangedFilesBetween() error = %v", err)
		}

		if len(files) != 2 {
			t.Errorf("Expected 2 changed files, got %d: %v", len(files), files)
		}
	})

	t.Run("filtered files", func(t *testing.T) {
		files, err := g.ChangedFilesBetween(hash1, "HEAD", []string{"file1.txt"})

		if err != nil {
			t.Fatalf("ChangedFilesBetween() error = %v", err)
		}

		if len(files) != 1 {
			t.Errorf("Expected 1 changed file, got %d: %v", len(files), files)
		}
		if files[0] != "file1.txt" {
			t.Errorf("Expected file1.txt, got %s", files[0])
		}
	})

	t.Run("no changes", func(t *testing.T) {
		files, err := g.ChangedFilesBetween("HEAD", "HEAD", nil)

		if err != nil {
			t.Fatalf("ChangedFilesBetween() error = %v", err)
		}

		if files != nil && len(files) != 0 {
			t.Errorf("Expected no changed files, got %v", files)
		}
	})
}

func TestGit_CommitsBetween(t *testing.T) {
	dir := setupGitRepo(t)
	hash1 := commitFile(t, dir, "file1.txt", "content1", "First commit")
	commitFile(t, dir, "file2.txt", "content2", "Second commit")
	commitFile(t, dir, "file3.txt", "content3", "Third commit")

	g := New(dir)

	commits, err := g.CommitsBetween(hash1, "HEAD", nil)
	if err != nil {
		t.Fatalf("CommitsBetween() error = %v", err)
	}

	if len(commits) != 2 {
		t.Errorf("Expected 2 commits, got %d: %v", len(commits), commits)
	}
}

func TestGit_CommitDetails(t *testing.T) {
	dir := setupGitRepo(t)
	hash1 := commitFile(t, dir, "file1.txt", "content1", "First commit")
	commitFile(t, dir, "file2.txt", "content2", "Second commit")

	g := New(dir)

	details, err := g.CommitDetails(hash1, "HEAD", nil)
	if err != nil {
		t.Fatalf("CommitDetails() error = %v", err)
	}

	if len(details) != 1 {
		t.Fatalf("Expected 1 commit detail, got %d", len(details))
	}

	if details[0].Subject != "Second commit" {
		t.Errorf("Subject = %s, want 'Second commit'", details[0].Subject)
	}
	if len(details[0].Hash) != 40 {
		t.Errorf("Hash should be full 40 chars, got %s", details[0].Hash)
	}
	if len(details[0].Short) < 7 {
		t.Errorf("Short should be at least 7 chars, got %s", details[0].Short)
	}
}

func TestGit_Diff(t *testing.T) {
	dir := setupGitRepo(t)
	hash1 := commitFile(t, dir, "file.txt", "line1\nline2\n", "First commit")
	commitFile(t, dir, "file.txt", "line1\nmodified\nline3\n", "Second commit")

	g := New(dir)

	diff, err := g.Diff(hash1, "HEAD", nil)
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}

	if diff == "" {
		t.Error("Diff should not be empty")
	}
	if !strings.Contains(diff, "-line2") {
		t.Error("Diff should show removed line")
	}
	if !strings.Contains(diff, "+modified") {
		t.Error("Diff should show added line")
	}
}

func TestGit_ShowCommitDiff(t *testing.T) {
	dir := setupGitRepo(t)
	commitFile(t, dir, "file.txt", "original", "First commit")
	hash2 := commitFile(t, dir, "file.txt", "modified", "Second commit")

	g := New(dir)

	diff, err := g.ShowCommitDiff(hash2, nil)
	if err != nil {
		t.Fatalf("ShowCommitDiff() error = %v", err)
	}

	if diff == "" {
		t.Error("ShowCommitDiff should not be empty")
	}
	if !strings.Contains(diff, "-original") {
		t.Error("Diff should show removed content")
	}
	if !strings.Contains(diff, "+modified") {
		t.Error("Diff should show added content")
	}
}

func TestGit_FilesChangedInCommit(t *testing.T) {
	dir := setupGitRepo(t)
	commitFile(t, dir, "file1.txt", "content1", "First commit")
	os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("content2"), 0644)
	os.WriteFile(filepath.Join(dir, "file3.txt"), []byte("content3"), 0644)

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = dir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Add multiple files")
	cmd.Dir = dir
	cmd.Run()
	cmd = exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = dir
	out, _ := cmd.Output()
	hash := strings.TrimSpace(string(out))

	g := New(dir)

	t.Run("all files in commit", func(t *testing.T) {
		files, err := g.FilesChangedInCommit(hash, nil)
		if err != nil {
			t.Fatalf("FilesChangedInCommit() error = %v", err)
		}

		if len(files) != 2 {
			t.Errorf("Expected 2 files, got %d: %v", len(files), files)
		}
	})

	t.Run("filtered files", func(t *testing.T) {
		files, err := g.FilesChangedInCommit(hash, []string{"file2.txt"})
		if err != nil {
			t.Fatalf("FilesChangedInCommit() error = %v", err)
		}

		if len(files) != 1 {
			t.Errorf("Expected 1 file, got %d: %v", len(files), files)
		}
	})
}
