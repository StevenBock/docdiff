package commands

// @doc README.md

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/StevenBock/docdiff/internal/git"
	"github.com/StevenBock/docdiff/internal/report"
	"github.com/StevenBock/docdiff/internal/scanner"
)

var ErrDocsNeedUpdate = fmt.Errorf("docs linked to changed files need updating")

var (
	checkStaged      bool
	checkFiles       []string
	checkJSON        bool
	checkNoBacklinks bool
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Show only docs affected by your current changes",
	Long: `Check which documentation is affected by your current changes, ignoring
unrelated stale docs across the rest of the repo.

By default it inspects working-tree changes (staged, unstaged, and untracked).
Use --staged for index-only changes, or --files to check an explicit set.

A doc is "updated" if it was edited alongside its linked source files, and
"needs update" if its linked files changed but the doc itself did not. The exit
code is non-zero when any affected doc needs updating, so it answers a single
question for agents: did I satisfy docs for my change?`,
	RunE: runCheck,
}

func init() {
	checkCmd.Flags().BoolVar(&checkStaged, "staged", false, "only consider staged (index) changes")
	checkCmd.Flags().StringSliceVar(&checkFiles, "files", nil, "check an explicit list of files instead of git changes")
	checkCmd.Flags().BoolVar(&checkJSON, "json", false, "output as JSON")
	checkCmd.Flags().BoolVar(&checkNoBacklinks, "no-backlinks", false, "hide missing back-link (hygiene) suggestions")
	rootCmd.AddCommand(checkCmd)
}

type checkResult struct {
	Doc             string                 `json:"doc"`
	Status          string                 `json:"status"` // "updated" | "needs update" | "ownership changed"
	DocInChangeset  bool                   `json:"doc_in_changeset"`
	ChangedFiles    []string               `json:"changed_files"`
	LinkedFileCount int                    `json:"linked_file_count"`
	Annotations     []annotationProvenance `json:"annotations,omitempty"`
	// Reasons carries structural-gate provenance (gate: structural docs).
	Reasons []string `json:"reasons,omitempty"`
}

type annotationProvenance struct {
	File  string    `json:"file"`
	Line  int       `json:"line,omitempty"`
	Kind  string    `json:"kind"`
	Scope string    `json:"scope,omitempty"`
	Hits  []provHit `json:"hits,omitempty"`
}

// provHit is a changed line range that overlapped a scoped annotation's owned
// region, with a short excerpt so update-vs-ack is decidable from check output.
type provHit struct {
	Start   int      `json:"start"`
	End     int      `json:"end"`
	Excerpt []string `json:"excerpt,omitempty"`
}

func runCheck(cmd *cobra.Command, args []string) error {
	g := git.New(rootDir)

	changed, source, err := changedSet(g)
	if err != nil {
		return err
	}

	inChange := make(map[string]bool, len(changed))
	for _, f := range changed {
		inChange[f] = true
	}

	// Changed line ranges per file, so scoped annotations only flag docs whose
	// owned region was actually touched. Only diff-backed modes have hunks;
	// --files has none, so it falls back to whole-file ownership.
	var hunks map[string][]git.LineRange
	switch source {
	case "working tree":
		hunks, _ = g.ChangedHunksSince("HEAD", false, changed, cfg.AnnotationTag)
	case "staged changes":
		hunks, _ = g.ChangedHunksSince("HEAD", true, changed, cfg.AnnotationTag)
	}

	// File-level annotation-only classification, BEFORE doc matching: a file
	// whose every changed hunk is annotation churn is an ownership-map change,
	// not a code change — it must not gate any doc (bare links included).
	// annOnlyFiles keeps an empty hunk entry so "annotation-only change" stays
	// distinguishable from "no hunk info" (untracked/--files paths keep the
	// conservative whole-file fallback).
	hunks, annOnlyFiles := splitAnnotationHunks(hunks)

	s := scanner.New(cfg, registry)
	scanResult, err := s.Scan(rootDir)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	acks, err := loadAcks(rootDir)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to load %s: %v\n", acksFile, err)
		acks = map[string]string{}
	}

	// `gate: structural` docs invert Item-1 polarity and are evaluated in a
	// separate pass (diff-backed modes only; --files keeps normal behavior).
	structural := cfg.StructuralDocs()
	diffBacked := source == "working tree" || source == "staged changes"

	results := make([]checkResult, 0)
	ownership := make([]checkResult, 0)
	affected := make(map[string]bool)
	warnings := make([]string, 0)
	seenWarnings := make(map[string]bool)
	for doc, files := range scanResult.FilesByDoc {
		if diffBacked && structural[doc] {
			continue // body edits never flag structural docs
		}
		var linkedChanged, annOnlyLinked []string
		hitsByFile := make(map[string][]scopeHit)
		for _, f := range files {
			if !inChange[f] {
				continue
			}
			if annOnlyFiles[f] {
				annOnlyLinked = append(annOnlyLinked, f)
				continue
			}
			if hit, hits := fileHitsDoc(scanResult.Annotations[f], doc, hunks); hit {
				linkedChanged = append(linkedChanged, f)
				if len(hits) > 0 {
					hitsByFile[f] = hits
				}
			}
		}
		if len(linkedChanged) > 0 && !inChange[doc] {
			linkedChanged = changedFilesSinceBaseline(g, doc, linkedChanged, source, acks, cmd.ErrOrStderr())
		}
		if len(linkedChanged) == 0 {
			// Only trigger (if any) was annotation churn: ownership map info,
			// never a doc-update requirement.
			if len(annOnlyLinked) > 0 {
				sort.Strings(annOnlyLinked)
				provenance := make([]annotationProvenance, 0)
				for _, f := range annOnlyLinked {
					provenance = append(provenance, provenanceForDoc(scanResult.Annotations[f], doc, f, nil)...)
				}
				ownership = append(ownership, checkResult{
					Doc:             doc,
					Status:          "ownership changed",
					DocInChangeset:  inChange[doc],
					ChangedFiles:    annOnlyLinked,
					LinkedFileCount: len(files),
					Annotations:     provenance,
				})
			}
			continue
		}
		sort.Strings(linkedChanged)
		affected[doc] = true

		status := "needs update"
		if inChange[doc] {
			status = "updated"
		}
		provenance := make([]annotationProvenance, 0)
		for _, f := range linkedChanged {
			provenance = append(provenance, provenanceForDoc(scanResult.Annotations[f], doc, f, hitsByFile[f])...)
			for _, warning := range annotationWarnings(scanResult.Annotations[f], doc) {
				if !seenWarnings[warning] {
					seenWarnings[warning] = true
					warnings = append(warnings, warning)
				}
			}
		}
		results = append(results, checkResult{
			Doc:             doc,
			Status:          status,
			DocInChangeset:  inChange[doc],
			ChangedFiles:    linkedChanged,
			LinkedFileCount: len(files),
			Annotations:     provenance,
		})
	}

	if diffBacked && len(structural) > 0 {
		staged := source == "staged changes"
		// No pathspec: a filter built from --name-only would omit the old
		// side of renames and break git's rename pairing.
		diffText, _ := g.DiffSince("HEAD", staged, nil)
		deltas := annotationDeltasByFile(diffText, cfg.AnnotationTag)
		nameStatus, _ := g.NameStatusSince("HEAD", staged, nil)
		var untracked []string
		if !staged {
			untracked, _ = g.UntrackedFiles(nil)
		}

		for doc := range structural {
			linked := make(map[string]bool)
			for _, f := range scanResult.FilesByDoc[doc] {
				linked[f] = true
			}
			trigFiles, reasons := structuralTriggers(doc, deltas, nameStatus, untracked, func(f string) bool { return linked[f] })
			if len(trigFiles) == 0 {
				continue
			}
			affected[doc] = true
			status := "needs update"
			if inChange[doc] {
				status = "updated"
			}
			results = append(results, checkResult{
				Doc:             doc,
				Status:          status,
				DocInChangeset:  inChange[doc],
				ChangedFiles:    uniqueSorted(trigFiles),
				LinkedFileCount: len(scanResult.FilesByDoc[doc]),
				Reasons:         reasons,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool { return results[i].Doc < results[j].Doc })
	sort.Slice(ownership, func(i, j int) bool { return ownership[i].Doc < ownership[j].Doc })
	sort.Strings(warnings)

	// Stale docs the current change doesn't explain. Split by whether HEAD's own
	// commit put them on review: committing makes the working tree clean, so
	// without this split `check` reports "nothing linked to your changes" while
	// the docs that very commit affected sit in the hidden count, labelled
	// "unrelated". Non-gating either way — it reports on an already-made commit.
	stale := computeStaleDocs(g, scanResult.FilesByDoc, cmd.ErrOrStderr())
	outstanding, unrelatedStale := partitionStale(g, scanResult, stale, affected)

	// Surface missing back-links for the docs this change touches, so they're
	// caught in the normal flow instead of a separate `report --undocumented`.
	// This is hygiene, not a blocker: it never gates the exit code.
	var undocRefs []scanner.UndocumentedRef
	if !checkNoBacklinks {
		for _, ref := range scanResult.UndocumentedRefs {
			if affected[ref.DocPath] {
				undocRefs = append(undocRefs, ref)
			}
		}
	}

	needsUpdate := 0
	for _, r := range results {
		if r.Status == "needs update" {
			needsUpdate++
		}
	}

	out := cmd.OutOrStdout()
	if checkJSON {
		writeCheckJSON(out, source, results, ownership, undocRefs, outstanding, unrelatedStale, needsUpdate, warnings)
	} else {
		if note := scopeNote(g, source, changed); note != "" {
			fmt.Fprintln(out, note)
		}
		writeCheckHuman(out, source, results, ownership, undocRefs, outstanding, unrelatedStale, needsUpdate, warnings)
	}

	if needsUpdate > 0 {
		return ErrDocsNeedUpdate
	}
	return nil
}

// changedSet resolves the set of changed source files and a label for it.
// The default source can be pinned per repo via `check: {scope: staged}` in
// the config; scope is never inferred from tree state (a check run before
// staging would silently narrow to an empty set).
func changedSet(g *git.Git) ([]string, string, error) {
	switch {
	case len(checkFiles) > 0:
		files := make([]string, 0, len(checkFiles))
		for _, f := range checkFiles {
			if filepath.IsAbs(f) {
				if rel, err := filepath.Rel(rootDir, f); err == nil {
					f = rel
				}
			}
			files = append(files, filepath.ToSlash(f))
		}
		return files, "files", nil
	case checkStaged || cfg.Check.Scope == "staged":
		files, err := g.StagedFiles()
		return files, "staged changes", err
	default:
		if s := cfg.Check.Scope; s != "" && s != "working-tree" {
			return nil, "", fmt.Errorf("invalid check.scope %q in config (want \"staged\" or \"working-tree\")", s)
		}
		files, err := g.WorkingTreeFiles()
		return files, "working tree", err
	}
}

// scopeNote describes files the chosen scope leaves out (staged mode) or
// silently includes (working-tree mode with staged changes). Info only.
func scopeNote(g *git.Git, source string, changed []string) string {
	switch source {
	case "staged changes":
		wt, err := g.WorkingTreeFiles()
		if err != nil {
			return ""
		}
		inScope := make(map[string]bool, len(changed))
		for _, f := range changed {
			inScope[f] = true
		}
		outside := 0
		for _, f := range wt {
			if !inScope[f] {
				outside++
			}
		}
		if outside > 0 {
			return fmt.Sprintf("Note: %d changed file(s) outside the index ignored (staged scope).", outside)
		}
	case "working tree":
		st, err := g.StagedFiles()
		if err != nil || len(st) == 0 {
			return ""
		}
		return fmt.Sprintf("Note: %d file(s) are staged; working-tree scope checks them together with unstaged changes (use --staged for the index alone).", len(st))
	}
	return ""
}

// partitionStale splits the stale docs the current change doesn't explain into
// those HEAD's own diff put on review ("outstanding from HEAD") and the rest,
// which are genuinely unrelated. Best-effort: if HEAD can't be diffed (no
// commits yet), everything falls back to unrelated.
func partitionStale(g *git.Git, scanResult *scanner.Result, stale map[string]*report.StaleDoc, affected map[string]bool) ([]string, int) {
	fromHead := make(map[string]bool)
	if docs, err := docsRequiredByHead(g, scanResult, stale); err == nil {
		for _, d := range docs {
			fromHead[d] = true
		}
	}

	outstanding := make([]string, 0)
	unrelated := 0
	for doc := range stale {
		switch {
		case affected[doc]: // already reported in a section above
		case fromHead[doc]:
			outstanding = append(outstanding, doc)
		default:
			unrelated++
		}
	}
	sort.Strings(outstanding)
	return outstanding, unrelated
}

func changedFilesSinceBaseline(g *git.Git, doc string, files []string, source string, acks map[string]string, errOut io.Writer) []string {
	if source == "working tree" {
		return files
	}

	baseline, err := baselineForDoc(g, doc, acks)
	if err != nil {
		fmt.Fprintf(errOut, "Warning: failed to find last commit for %s: %v\n", doc, err)
		return files
	}
	if baseline.Effective == "" {
		return files
	}

	staged := source == "staged changes"
	changed, err := g.ChangedFilesSince(baseline.Effective, staged, files)
	if err != nil {
		fmt.Fprintf(errOut, "Warning: failed to check changes for %s (%s..%s): %v\n", doc, baseline.Effective, source, err)
		return files
	}
	if staged && len(changed) > 0 {
		// Drop files whose cumulative diff since the doc's baseline is pure
		// annotation churn — there is nothing for the doc to document.
		if diff, derr := g.DiffSince(baseline.Effective, true, changed); derr == nil {
			annOnly := annotationOnlyDiffFiles(diff, cfg.AnnotationTag)
			kept := changed[:0]
			for _, f := range changed {
				if !annOnly[f] {
					kept = append(kept, f)
				}
			}
			changed = kept
		}
	}
	if source == "files" {
		untracked, err := g.UntrackedFiles(files)
		if err == nil {
			changed = append(changed, untracked...)
		}
	}

	changedSet := make(map[string]bool, len(changed))
	for _, f := range changed {
		changedSet[f] = true
	}
	filtered := make([]string, 0, len(files))
	for _, f := range files {
		if changedSet[f] {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

func writeCheckHuman(out io.Writer, source string, results []checkResult, ownership []checkResult, undocRefs []scanner.UndocumentedRef, outstanding []string, unrelatedStale, needsUpdate int, warnings []string) {
	// "staged changes" + the literal word "changes" in the templates below
	// would print "staged changes changes".
	source = strings.TrimSuffix(source, " changes")
	if len(results) == 0 && len(ownership) == 0 {
		fmt.Fprintf(out, "No docs are linked to your %s changes.\n", source)
		writeOutstanding(out, outstanding)
		if unrelatedStale > 0 {
			fmt.Fprintf(out, "(%d unrelated stale docs hidden — run 'docdiff report' to see them.)\n", unrelatedStale)
		}
		return
	}

	if len(results) > 0 {
		// Section 1 — required: docs whose linked code changed but weren't
		// edited. This is the only section that drives the exit code.
		fmt.Fprintf(out, "Required for %s changes (%d):\n", source, needsUpdate)
		if needsUpdate == 0 {
			fmt.Fprintln(out, "  none — every affected doc was edited alongside its code.")
		}
		for _, r := range results {
			if r.Status == "needs update" {
				fmt.Fprintf(out, "  %s: needs update%s\n", r.Doc, broadHint(r))
				writeProvenance(out, r)
			}
		}

		// Section 2 — already satisfied: edited in this changeset. Informational.
		updated := len(results) - needsUpdate
		if updated > 0 {
			fmt.Fprintf(out, "\nAlready updated in this changeset (%d):\n", updated)
			for _, r := range results {
				if r.Status == "updated" {
					fmt.Fprintf(out, "  %s%s\n", r.Doc, broadHint(r))
					writeProvenance(out, r)
				}
			}
		}
	} else {
		fmt.Fprintf(out, "No doc updates required for your %s changes.\n", source)
	}

	// Ownership-map info: docs whose only trigger was annotation churn.
	// Informational — never gates the exit code.
	if len(ownership) > 0 {
		fmt.Fprintf(out, "\nOwnership map changed (%d):\n", len(ownership))
		for _, r := range ownership {
			fmt.Fprintf(out, "  %s: annotation-only change — no doc update required\n", r.Doc)
			writeProvenance(out, r)
		}
	}

	if len(warnings) > 0 {
		fmt.Fprintf(out, "\nAnnotation warnings:\n")
		for _, warning := range warnings {
			fmt.Fprintf(out, "  %s\n", warning)
		}
	}

	// Section 3 — hygiene: missing back-links. Never gates the exit code.
	if len(undocRefs) > 0 {
		fmt.Fprintf(out, "\nBack-link hygiene — optional (%d):\n", len(undocRefs))
		for _, ref := range undocRefs {
			fmt.Fprintf(out, "  %s references %s (add: %s %s)\n", ref.DocPath, ref.SourceFile, cfg.AnnotationTag, ref.DocPath)
		}
	}

	writeOutstanding(out, outstanding)

	if unrelatedStale > 0 {
		fmt.Fprintf(out, "\nUnrelated stale docs: %d hidden (run 'docdiff report' for the full picture).\n", unrelatedStale)
	}

	if needsUpdate > 0 {
		fmt.Fprintln(out, "\nNext: update the required docs above and commit them together with your code —")
		fmt.Fprintln(out, "editing a doc in the same commit as its linked source marks it reviewed.")
		fmt.Fprintln(out, "New to docdiff? Run 'docdiff onboard' for agent-ready workflow instructions.")
	}
}

// writeOutstanding names the docs HEAD's own commit put on review, and the
// commands that clear them. This section exists for the agent that committed,
// ran check, saw green and stopped — so a bare count is not enough; it has to
// say what to do next.
func writeOutstanding(out io.Writer, outstanding []string) {
	if len(outstanding) == 0 {
		return
	}
	fmt.Fprintf(out, "\nOutstanding from HEAD (%d) — your last commit changed code these docs cover:\n", len(outstanding))
	for _, doc := range outstanding {
		fmt.Fprintf(out, "  %s\n", doc)
	}
	fmt.Fprintln(out, "  Next: edit any that are now wrong and fold them in with 'git commit --amend',")
	fmt.Fprintln(out, "  then record the rest as reviewed-needing-no-edit:")
	fmt.Fprintln(out, "    docdiff ack --required --amend")
}

const broadDocLinkedFileThreshold = 20

func broadHint(r checkResult) string {
	if r.LinkedFileCount >= broadDocLinkedFileThreshold {
		return fmt.Sprintf(" (broad: %d linked files)", r.LinkedFileCount)
	}
	return ""
}

func writeProvenance(out io.Writer, r checkResult) {
	for _, reason := range r.Reasons {
		fmt.Fprintf(out, "    %s\n", reason)
	}
	for _, p := range r.Annotations {
		switch {
		case p.Kind == "scoped":
			fmt.Fprintf(out, "    via %s:%d scoped %s #%s\n", p.File, p.Line, cfg.AnnotationTag, p.Scope)
			for _, h := range p.Hits {
				fmt.Fprintf(out, "      lines %d–%d hit #%s\n", h.Start, h.End, p.Scope)
				for _, ln := range h.Excerpt {
					fmt.Fprintf(out, "        > %s\n", ln)
				}
			}
		case p.Line > 0:
			fmt.Fprintf(out, "    via %s:%d whole-file %s\n", p.File, p.Line, cfg.AnnotationTag)
		default:
			fmt.Fprintf(out, "    via %s %s\n", p.File, p.Kind)
		}
	}
}

const hitExcerptMaxLines = 3

func provenanceForDoc(ann *scanner.Annotation, doc, file string, hits []scopeHit) []annotationProvenance {
	if ann == nil {
		return []annotationProvenance{{File: file, Kind: "unknown annotation"}}
	}

	var fileLines []string // lazily loaded for excerpts
	excerpt := func(start, end int) []string {
		if fileLines == nil {
			data, err := os.ReadFile(filepath.Join(rootDir, file))
			if err != nil {
				return nil
			}
			fileLines = strings.Split(string(data), "\n")
		}
		if end > start+hitExcerptMaxLines-1 {
			end = start + hitExcerptMaxLines - 1
		}
		var out []string
		for i := start; i <= end && i <= len(fileLines); i++ {
			if i >= 1 {
				out = append(out, fileLines[i-1])
			}
		}
		return out
	}

	out := make([]annotationProvenance, 0)
	for _, d := range ann.Details {
		if d.Path != doc {
			continue
		}
		kind := "whole-file"
		if d.Scope != "" {
			kind = "scoped"
		}
		p := annotationProvenance{
			File:  file,
			Line:  d.Line,
			Kind:  kind,
			Scope: d.Scope,
		}
		for _, h := range hits {
			if h.AnnLine == d.Line {
				p.Hits = append(p.Hits, provHit{Start: h.Start, End: h.End, Excerpt: excerpt(h.Start, h.End)})
			}
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return []annotationProvenance{{File: file, Kind: "unknown annotation"}}
	}
	return out
}

func annotationWarnings(ann *scanner.Annotation, doc string) []string {
	if ann == nil {
		return nil
	}
	hasWholeFile := false
	hasScoped := false
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
		return []string{fmt.Sprintf("%s mixes whole-file and scoped annotations for %s; whole-file ownership wins for that doc.", ann.FilePath, doc)}
	}
	return nil
}

func writeCheckJSON(out io.Writer, source string, results []checkResult, ownership []checkResult, undocRefs []scanner.UndocumentedRef, outstanding []string, unrelatedStale, needsUpdate int, warnings []string) {
	if undocRefs == nil {
		undocRefs = []scanner.UndocumentedRef{}
	}
	if warnings == nil {
		warnings = []string{}
	}
	if outstanding == nil {
		outstanding = []string{}
	}
	payload := struct {
		Source              string                    `json:"source"`
		Affected            []checkResult             `json:"affected"`
		OwnershipChanged    []checkResult             `json:"ownership_changed"`
		UndocumentedRefs    []scanner.UndocumentedRef `json:"undocumented_refs"`
		NeedsUpdate         int                       `json:"needs_update"`
		OutstandingFromHead []string                  `json:"outstanding_from_head"`
		UnrelatedStale      int                       `json:"unrelated_stale"`
		Warnings            []string                  `json:"warnings"`
	}{
		Source:              source,
		Affected:            results,
		OwnershipChanged:    ownership,
		UndocumentedRefs:    undocRefs,
		NeedsUpdate:         needsUpdate,
		OutstandingFromHead: outstanding,
		UnrelatedStale:      unrelatedStale,
		Warnings:            warnings,
	}
	data, _ := json.MarshalIndent(payload, "", "  ")
	fmt.Fprintln(out, string(data))
}

// splitAnnotationHunks drops AnnotationOnly ranges from the hunk map. A file
// whose every hunk was annotation-only keeps an EMPTY entry (so scoped matching
// sees "no touched lines", distinct from "no hunk info") and is reported in the
// annotation-only set.
func splitAnnotationHunks(hunks map[string][]git.LineRange) (map[string][]git.LineRange, map[string]bool) {
	if hunks == nil {
		return nil, nil
	}
	filtered := make(map[string][]git.LineRange, len(hunks))
	annOnly := make(map[string]bool)
	for f, ranges := range hunks {
		real := make([]git.LineRange, 0, len(ranges))
		for _, r := range ranges {
			if !r.AnnotationOnly {
				real = append(real, r)
			}
		}
		filtered[f] = real
		if len(real) == 0 && len(ranges) > 0 {
			annOnly[f] = true
		}
	}
	return filtered, annOnly
}
