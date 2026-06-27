package commands

import "testing"

func TestSuggestDoc(t *testing.T) {
	votes := map[string]map[string]int{
		"internal/api": {"docs/API.md": 3, "docs/Other.md": 1},
		"internal/db":  {"docs/DB.md": 2},
	}

	cases := map[string]string{
		"internal/api/new.go":      "docs/API.md", // direct dir, majority wins
		"internal/api/sub/deep.go": "docs/API.md", // ancestor dir votes
		"internal/db/conn.go":      "docs/DB.md",
		"cmd/tool/main.go":         "", // no annotated neighbor
		"toplevel.go":              "", // no votes at "."
	}

	for file, want := range cases {
		if got := suggestDoc(file, votes); got != want {
			t.Errorf("suggestDoc(%q) = %q, want %q", file, got, want)
		}
	}
}

func TestCommentToken(t *testing.T) {
	for file, want := range map[string]string{
		"a.go": "//", "b.py": "#", "c.rb": "#", "d.sh": "#", "e.unknown": "//",
	} {
		if got := commentToken(file); got != want {
			t.Errorf("commentToken(%q) = %q, want %q", file, got, want)
		}
	}
}
