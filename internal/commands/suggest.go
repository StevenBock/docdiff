package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/StevenBock/docdiff/internal/scanner"
)

var suggestJSON bool

var suggestCmd = &cobra.Command{
	Use:   "suggest",
	Short: "Suggest @doc annotations for orphaned files, grouped by likely owning doc",
	Long: `Group orphaned files (no @doc annotation) by the documentation file that
their already-annotated neighbors point to, and emit ready-to-paste annotation
lines in batches.

The owner is inferred by directory: for each orphaned file, the nearest ancestor
directory that contains annotated files votes for its most common doc. Files with
no annotated neighbors are listed separately as unmapped.`,
	RunE: runSuggest,
}

func init() {
	suggestCmd.Flags().BoolVar(&suggestJSON, "json", false, "output as JSON")
	rootCmd.AddCommand(suggestCmd)
}

type docSuggestion struct {
	Doc   string   `json:"doc"`
	Files []string `json:"files"`
}

func runSuggest(cmd *cobra.Command, args []string) error {
	s := scanner.New(cfg, registry)
	scanResult, err := s.Scan(rootDir)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	// Tally, per directory, which docs the annotated files in it point to.
	// ponytail: directory-vote heuristic, no content analysis. Upgrade to
	// import-graph ownership only if path proximity proves too coarse.
	votes := make(map[string]map[string]int) // dir -> doc -> count
	for file, ann := range scanResult.Annotations {
		dir := filepath.ToSlash(filepath.Dir(file))
		for _, doc := range ann.DocPaths {
			if votes[dir] == nil {
				votes[dir] = make(map[string]int)
			}
			votes[dir][doc]++
		}
	}

	byDoc := make(map[string][]string)
	var unmapped []string
	for _, file := range scanResult.OrphanedFiles() {
		if doc := suggestDoc(file, votes); doc != "" {
			byDoc[doc] = append(byDoc[doc], file)
		} else {
			unmapped = append(unmapped, file)
		}
	}

	docs := make([]string, 0, len(byDoc))
	for doc := range byDoc {
		docs = append(docs, doc)
	}
	sort.Strings(docs)

	suggestions := make([]docSuggestion, 0, len(byDoc))
	for _, doc := range docs {
		files := byDoc[doc]
		sort.Strings(files)
		suggestions = append(suggestions, docSuggestion{Doc: doc, Files: files})
	}
	sort.Strings(unmapped)

	out := cmd.OutOrStdout()
	if suggestJSON {
		return writeSuggestJSON(out, suggestions, unmapped)
	}
	writeSuggestHuman(out, suggestions, unmapped)
	return nil
}

// suggestDoc walks up file's directory ancestry and returns the top-voted doc
// from the nearest ancestor directory that has any votes.
func suggestDoc(file string, votes map[string]map[string]int) string {
	dir := filepath.ToSlash(filepath.Dir(file))
	for {
		if docs := votes[dir]; docs != nil {
			best, bestN := "", 0
			for doc, n := range docs {
				if n > bestN || (n == bestN && doc < best) {
					best, bestN = doc, n
				}
			}
			return best
		}
		parent := filepath.ToSlash(filepath.Dir(dir))
		if parent == dir || dir == "." {
			return ""
		}
		dir = parent
	}
}

func writeSuggestHuman(out io.Writer, suggestions []docSuggestion, unmapped []string) {
	total := 0
	for _, s := range suggestions {
		total += len(s.Files)
	}
	if total == 0 && len(unmapped) == 0 {
		fmt.Fprintf(out, "No orphaned files. All source files have %s annotations.\n", cfg.AnnotationTag)
		return
	}

	fmt.Fprintf(out, "Suggested %s annotations for %d orphaned file(s):\n\n", cfg.AnnotationTag, total)
	for _, s := range suggestions {
		fmt.Fprintf(out, "%s (%d files):\n", s.Doc, len(s.Files))
		for _, f := range s.Files {
			fmt.Fprintf(out, "  %s %s %s   ->   %s\n", commentToken(f), cfg.AnnotationTag, s.Doc, f)
		}
		fmt.Fprintln(out)
	}

	if len(unmapped) > 0 {
		fmt.Fprintf(out, "Unmapped (no annotated neighbor) — %d file(s):\n", len(unmapped))
		for _, f := range unmapped {
			fmt.Fprintf(out, "  %s\n", f)
		}
		fmt.Fprintln(out)
	}

	fmt.Fprintln(out, "Add each annotation to the file's top comment, then run 'docdiff sync --affected --to HEAD' after committing.")
}

func writeSuggestJSON(out io.Writer, suggestions []docSuggestion, unmapped []string) error {
	if unmapped == nil {
		unmapped = []string{}
	}
	payload := struct {
		Suggestions []docSuggestion `json:"suggestions"`
		Unmapped    []string        `json:"unmapped"`
	}{
		Suggestions: suggestions,
		Unmapped:    unmapped,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(out, string(data))
	return nil
}

// commentToken returns the line-comment prefix for a file by extension,
// defaulting to "//" for the C-family majority.
func commentToken(file string) string {
	switch strings.ToLower(filepath.Ext(file)) {
	case ".py", ".rb", ".sh", ".bash", ".zsh", ".yaml", ".yml", ".ps1", ".pl", ".r":
		return "#"
	default:
		return "//"
	}
}
