package commands

import "strings"

import "testing"

func TestFilterAnnotationDiff(t *testing.T) {
	diff := strings.Join([]string{
		"diff --git a/foo.go b/foo.go",
		"index 111..222 100644",
		"--- a/foo.go",
		"+++ b/foo.go",
		"@@ -1,2 +1,3 @@",
		" package foo",
		"+// @doc docs/Foo.md",
		" func A() {}",
		"@@ -10,1 +11,1 @@",
		"-func B() {}",
		"+func B(x int) {}",
		"diff --git a/bar.go b/bar.go",
		"index 333..444 100644",
		"--- a/bar.go",
		"+++ b/bar.go",
		"@@ -1,1 +1,2 @@",
		" package bar",
		"+// @doc docs/Bar.md",
	}, "\n")

	got := filterAnnotationDiff(diff, "@doc")

	// foo.go: annotation-only hunk dropped, real-code hunk kept.
	if strings.Contains(got, "@doc docs/Foo.md") {
		t.Errorf("annotation-only hunk in foo.go was not dropped:\n%s", got)
	}
	if !strings.Contains(got, "func B(x int)") {
		t.Errorf("real code hunk was lost:\n%s", got)
	}
	// bar.go: every hunk annotation-only -> whole file block dropped.
	if strings.Contains(got, "bar.go") {
		t.Errorf("annotation-only file block was not dropped:\n%s", got)
	}
}

func TestAnnotationOnlyDiffFiles(t *testing.T) {
	diff := strings.Join([]string{
		"diff --git a/pure.go b/pure.go",
		"--- a/pure.go",
		"+++ b/pure.go",
		"@@ -2 +2 @@",
		"-// @doc docs/A.md",
		"+// @doc docs/B.md",
		"diff --git a/mixed.go b/mixed.go",
		"--- a/mixed.go",
		"+++ b/mixed.go",
		"@@ -2 +2 @@",
		"-// @doc docs/A.md",
		"+// @doc docs/B.md",
		"@@ -9 +9 @@",
		"-old()",
		"+new()",
		"diff --git a/gone.go b/gone.go",
		"--- a/gone.go",
		"+++ /dev/null",
		"@@ -1,2 +0,0 @@",
		"-// @doc docs/A.md",
		"-func Gone() {}",
	}, "\n")

	got := annotationOnlyDiffFiles(diff, "@doc")
	if !got["pure.go"] {
		t.Error("pure.go should be annotation-only")
	}
	if got["mixed.go"] {
		t.Error("mixed.go has a real hunk and must not be annotation-only")
	}
	if got["gone.go"] {
		t.Error("a deleted file's body removal is a real change")
	}
}

func TestAnnotationOnlyHunk(t *testing.T) {
	mixed := []string{"@@ -1 +1 @@", "-old", "+// @doc x.md"}
	if annotationOnlyHunk(mixed, "@doc") {
		t.Error("hunk with a non-annotation change should be kept")
	}
	pure := []string{"@@ -1 +1,2 @@", " ctx", "+// @doc x.md"}
	if !annotationOnlyHunk(pure, "@doc") {
		t.Error("hunk that only adds an annotation should be annotation-only")
	}
	noChange := []string{"@@ -1 +1 @@", " ctx"}
	if annotationOnlyHunk(noChange, "@doc") {
		t.Error("hunk with no changed lines is not annotation-only")
	}
}
