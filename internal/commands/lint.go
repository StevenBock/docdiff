package commands

// @doc README.md

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/StevenBock/docdiff/internal/language"
	"github.com/StevenBock/docdiff/internal/scanner"
)

var ErrLintProblems = fmt.Errorf("annotation lint found problems")

var lintJSON bool

var lintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Lint annotation hygiene across the repo",
	Long: `Check the health of the annotation map itself, independent of staleness.

Problems (non-zero exit):
  scope-to-eof-unbounded
                   a scoped annotation runs to end of file with a real region
                   above it, so how far it reaches is a live question. Put the
                   terminator (tag + '-') on the LAST line to mark that as
                   deliberate, or earlier if the scope shouldn't own everything
                   below — both clear it. Terminating is worth preferring on its
                   own merits: code appended later then lands outside the region
                   and shows up as orphaned, instead of being silently absorbed
                   into whatever scope reached the end
  mixed-ownership  a file links the same doc both whole-file and scoped;
                   whole-file wins, the scopes are dead
  fan-out          one file whole-file-links many docs, so every change flags
                   them all (threshold: lint.fan_out_threshold, default 3)

  scope-drift      a doc that uses '<!-- @scope name -->' section markers has
                   markers with no matching code scope, or code scopes with no
                   marker (only docs with at least one marker are checked)

  unlinked         a doc names neither the file, its directory, nor its bare
                   filename, yet the file links to the doc — nothing routes a
                   task from the doc to the file

Advisory (never affects the exit code):
  scope-covers-file
                   a scoped annotation covers its whole file — nothing above it
                   owns any code, so the scope name narrows nothing. Usually
                   deliberate, but a scope that was written at the top and never
                   bounded looks identical from here, so it stays visible rather
                   than being silenced. A file whose ONLY annotation is scoped
                   is not reported at all: it owns the whole file either way
  dir-only         the doc names the file's directory but not the file
  basename-only    the doc names the bare filename but not its path`,
	RunE: runLint,
}

func init() {
	lintCmd.Flags().BoolVar(&lintJSON, "json", false, "output as JSON")
	rootCmd.AddCommand(lintCmd)
}

type lintFinding struct {
	Rule     string `json:"rule"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Doc      string `json:"doc,omitempty"`
	Message  string `json:"message"`
	Advisory bool   `json:"advisory"`
}

func runLint(cmd *cobra.Command, args []string) error {
	s := scanner.New(cfg, registry)
	scanResult, err := s.Scan(rootDir)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	findings := lintFindings(scanResult, cfg.AnnotationTag, cfg.LintFanOutThreshold(), fileBoundaryAbove(scanResult))

	out := cmd.OutOrStdout()
	problems := 0
	for _, f := range findings {
		if !f.Advisory {
			problems++
		}
	}

	if lintJSON {
		// Revision identifies which build produced these counts. A consumer
		// diffing `problems` across versions can already attribute the change
		// per rule (every finding carries one); this says which docdiff decided.
		payload := struct {
			Findings []lintFinding `json:"findings"`
			Problems int           `json:"problems"`
			Revision string        `json:"revision,omitempty"`
		}{Findings: findings, Problems: problems, Revision: buildRevision()}
		data, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Fprintln(out, string(data))
	} else {
		writeLintHuman(out, findings, problems)
	}

	if problems > 0 {
		return ErrLintProblems
	}
	return nil
}

func writeLintHuman(out io.Writer, findings []lintFinding, problems int) {
	if len(findings) == 0 {
		fmt.Fprintln(out, "No lint findings.")
		return
	}
	if problems > 0 {
		fmt.Fprintf(out, "Problems (%d):\n", problems)
		for _, f := range findings {
			if !f.Advisory {
				fmt.Fprintf(out, "  [%s] %s\n", f.Rule, f.Message)
			}
		}
	}
	if advisories := len(findings) - problems; advisories > 0 {
		if problems > 0 {
			fmt.Fprintln(out)
		}
		fmt.Fprintf(out, "Advisory (%d):\n", advisories)
		for _, f := range findings {
			if f.Advisory {
				fmt.Fprintf(out, "  [%s] %s\n", f.Rule, f.Message)
			}
		}
	}
}

// boundaryAbove reports whether any annotation preceding annLine in file owns
// substantive code. nil means "cannot tell" — every scope-to-eof then gates,
// which is the conservative direction: a finding that should have been advisory
// is noise, one that should have gated and didn't is a defect nobody sees.
type boundaryAbove func(file string, annLine int) bool

func lintFindings(res *scanner.Result, tag string, fanOutThreshold int, hasBoundaryAbove boundaryAbove) []lintFinding {
	var out []lintFinding

	for file, ann := range res.Annotations {
		// scope-to-eof: a scoped region that runs to end of file AFTER an earlier
		// annotation. Only then is "where does this region stop" a real question
		// — a file whose single annotation is scoped owns everything below it,
		// exactly as a whole-file link would, and needs no terminator.
		if len(ann.Details) > 1 {
			if d := ann.Details[len(ann.Details)-1]; d.Scope != "" && d.Path != language.Terminator {
				// Two findings, not one at two volumes. If something above owns
				// code, a boundary exists and the region's extent is a real
				// question — that gates. If nothing does, the scope covers its
				// whole file, which is usually deliberate but is sometimes a
				// scope that was never bounded; those look identical from here,
				// so it stays visible as an advisory rather than being silenced.
				if hasBoundaryAbove != nil && !hasBoundaryAbove(file, d.Line) {
					out = append(out, lintFinding{
						Rule:     "scope-covers-file",
						File:     file,
						Line:     d.Line,
						Doc:      d.Path,
						Advisory: true,
						Message: fmt.Sprintf("%s:%d: scoped annotation #%s covers the whole file — nothing above it owns any code, so the scope name is doing no narrowing. Re-link whole-file, or bound it with '%s -' if it was meant to cover less",
							file, d.Line, d.Scope, tag),
					})
				} else {
					// Name BOTH remedies. Saying only "if it shouldn't own
					// everything below" tells an author whose tail genuinely is
					// theirs that the fix isn't for them, so they conclude there
					// is nothing to do — which is how a correct finding becomes
					// a permanently unactionable one.
					out = append(out, lintFinding{
						Rule: "scope-to-eof-unbounded",
						File: file,
						Line: d.Line,
						Doc:  d.Path,
						Message: fmt.Sprintf("%s:%d: scoped annotation #%s is the last of %d in the file, so its region runs to end of file — put '%s -' on the last line to mark that deliberate, or earlier if it shouldn't own everything below",
							file, d.Line, d.Scope, len(ann.Details), tag),
					})
				}
			}
		}

		// mixed-ownership: same doc linked both whole-file and scoped.
		for _, doc := range ann.DocPaths {
			hasWholeFile, hasScoped := false, false
			for _, d := range ann.Details {
				if d.Path != doc {
					continue
				}
				if d.Scope == "" {
					hasWholeFile = true
				} else {
					hasScoped = true
				}
			}
			if hasWholeFile && hasScoped {
				out = append(out, lintFinding{
					Rule: "mixed-ownership",
					File: file,
					Doc:  doc,
					Message: fmt.Sprintf("%s links %s both whole-file and scoped; whole-file wins and the scopes are dead — scope every link to %s, or drop the scoped ones",
						file, doc, doc),
				})
			}
		}

		// fan-out: many distinct whole-file links on one file.
		bare := make(map[string]bool)
		for _, d := range ann.Details {
			if d.Scope == "" && d.Path != language.Terminator {
				bare[d.Path] = true
			}
		}
		if len(bare) >= fanOutThreshold {
			docs := make([]string, 0, len(bare))
			for d := range bare {
				docs = append(docs, d)
			}
			sort.Strings(docs)
			out = append(out, lintFinding{
				Rule: "fan-out",
				File: file,
				Message: fmt.Sprintf("%s whole-file-links %d docs (%s); any change flags them all — scope the links (#name) or split the file",
					file, len(docs), strings.Join(docs, ", ")),
			})
		}
	}

	// scope-drift: doc-side section markers (<!-- @scope name -->) vs code
	// annotation scopes, both directions. Markers are opt-in per doc: a doc
	// with no markers is never checked, but once it uses them, keeping them
	// matched to code scopes is the contract.
	scopeLocs := make(map[string]map[string]string) // doc -> scope -> "file:line"
	for file, ann := range res.Annotations {
		for _, d := range ann.Details {
			if d.Scope == "" || d.Path == language.Terminator {
				continue
			}
			if scopeLocs[d.Path] == nil {
				scopeLocs[d.Path] = make(map[string]string)
			}
			if _, ok := scopeLocs[d.Path][d.Scope]; !ok {
				scopeLocs[d.Path][d.Scope] = fmt.Sprintf("%s:%d", file, d.Line)
			}
		}
	}
	for doc, markers := range res.DocScopeMarkers {
		if len(markers) == 0 {
			continue
		}
		markerSet := make(map[string]bool, len(markers))
		for _, m := range markers {
			markerSet[m] = true
			if _, ok := scopeLocs[doc][m]; !ok {
				out = append(out, lintFinding{
					Rule: "scope-drift",
					Doc:  doc,
					Message: fmt.Sprintf("%s marks section #%s but no code annotation targets it with that scope — the scope was renamed or removed; update or delete the marker",
						doc, m),
				})
			}
		}
		scopes := make([]string, 0, len(scopeLocs[doc]))
		for s := range scopeLocs[doc] {
			scopes = append(scopes, s)
		}
		sort.Strings(scopes)
		for _, s := range scopes {
			if !markerSet[s] {
				out = append(out, lintFinding{
					Rule: "scope-drift",
					Doc:  doc,
					File: scopeLocs[doc][s],
					Message: fmt.Sprintf("code scope #%s (%s) has no '<!-- @scope %s -->' marker in %s — add the marker or rename the scope",
						s, scopeLocs[doc][s], s, doc),
				})
			}
		}
	}

	// Back-link coverage, tiered by how well the doc actually routes a reader to
	// the file. Only "unlinked" gates: a doc that names the directory or the
	// bare filename is weaker coverage, not a broken link, and gating on those
	// is what gets the whole check muted.
	for doc, files := range res.FilesByDoc {
		refs, read := res.DocRefs[doc]
		if !read {
			continue
		}
		weak := res.DocWeakRefs[doc]
		for _, f := range files {
			if refs[f] {
				continue
			}
			switch {
			case weak.Basenames[path.Base(f)]:
				out = append(out, lintFinding{
					Rule:     "basename-only",
					File:     f,
					Doc:      doc,
					Advisory: true,
					Message: fmt.Sprintf("%s names %s but not its path — an agent grepping for %s finds the doc, one navigating from the doc has to guess the directory",
						doc, path.Base(f), path.Base(f)),
				})
			case weak.Dirs[path.Dir(f)]:
				out = append(out, lintFinding{
					Rule:     "dir-only",
					File:     f,
					Doc:      doc,
					Advisory: true,
					Message: fmt.Sprintf("%s names %s/ but not %s — the doc routes to the directory, not to the file",
						doc, path.Dir(f), f),
				})
			default:
				out = append(out, lintFinding{
					Rule: "unlinked",
					File: f,
					Doc:  doc,
					Message: fmt.Sprintf("%s never mentions %s, its directory, or its filename, yet %s links to it — the doc cannot route a task to this file",
						doc, f, f),
				})
			}
		}
	}

	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Advisory != b.Advisory {
			return !a.Advisory
		}
		if a.Rule != b.Rule {
			return a.Rule < b.Rule
		}
		if a.File != b.File {
			return a.File < b.File
		}
		return a.Doc < b.Doc
	})
	return out
}

// fileBoundaryAbove builds the boundaryAbove predicate over real files, reading
// each one at most once. A file that cannot be read or whose language is not
// registered reports true, so its finding keeps gating rather than being quietly
// demoted on an I/O error.
func fileBoundaryAbove(res *scanner.Result) boundaryAbove {
	cache := make(map[string]bool)
	return func(file string, annLine int) bool {
		key := fmt.Sprintf("%s\x00%d", file, annLine)
		if v, ok := cache[key]; ok {
			return v
		}
		v := true
		if ann := res.Annotations[file]; ann != nil {
			if strat, ok := registry.GetByExtension(filepath.Ext(file)); ok {
				if content, err := os.ReadFile(filepath.Join(rootDir, file)); err == nil {
					v = regionOwnsCodeAbove(content, strat.CommentPatterns(), ann.Details, annLine)
				}
			}
		}
		cache[key] = v
		return v
	}
}
