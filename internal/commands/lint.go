package commands

// @doc README.md

import (
	"encoding/json"
	"fmt"
	"io"
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
  scope-to-eof     a file's only annotation is scoped, so its region silently
                   runs to end of file — close it with the terminator form
                   (tag + '-') or use a whole-file link
  mixed-ownership  a file links the same doc both whole-file and scoped;
                   whole-file wins, the scopes are dead
  fan-out          one file whole-file-links many docs, so every change flags
                   them all (threshold: lint.fan_out_threshold, default 3)

  scope-drift      a doc that uses '<!-- @scope name -->' section markers has
                   markers with no matching code scope, or code scopes with no
                   marker (only docs with at least one marker are checked)

Advisory (never affects the exit code):
  vestigial-link   a doc never mentions a file that links to it — the link may
                   be vestigial, or the doc is genuinely high-level`,
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

	findings := lintFindings(scanResult, cfg.AnnotationTag, cfg.LintFanOutThreshold())

	out := cmd.OutOrStdout()
	problems := 0
	for _, f := range findings {
		if !f.Advisory {
			problems++
		}
	}

	if lintJSON {
		payload := struct {
			Findings []lintFinding `json:"findings"`
			Problems int           `json:"problems"`
		}{Findings: findings, Problems: problems}
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

func lintFindings(res *scanner.Result, tag string, fanOutThreshold int) []lintFinding {
	var out []lintFinding

	for file, ann := range res.Annotations {
		// scope-to-eof: the file's only annotation is scoped, so its "narrow"
		// region actually owns everything to end of file.
		if len(ann.Details) == 1 && ann.Details[0].Scope != "" {
			d := ann.Details[0]
			out = append(out, lintFinding{
				Rule: "scope-to-eof",
				File: file,
				Line: d.Line,
				Doc:  d.Path,
				Message: fmt.Sprintf("%s:%d: scoped annotation #%s is the file's only annotation, so its region runs to end of file — close it with '%s -' or use a whole-file link",
					file, d.Line, d.Scope, tag),
			})
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

	// vestigial-link: the doc was read as content and never mentions a file
	// that links to it. Advisory only — plenty of good docs are high-level.
	for doc, files := range res.FilesByDoc {
		refs, read := res.DocRefs[doc]
		if !read {
			continue
		}
		for _, f := range files {
			if !refs[f] {
				out = append(out, lintFinding{
					Rule:     "vestigial-link",
					File:     f,
					Doc:      doc,
					Advisory: true,
					Message:  fmt.Sprintf("%s never mentions %s, yet %s links to it — the link may be vestigial", doc, f, f),
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
