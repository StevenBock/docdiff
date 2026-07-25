package commands

// @doc CLAUDE.md

import (
	"fmt"
	"io"
	"regexp"
	"strconv"

	"github.com/StevenBock/docdiff/internal/git"
	"github.com/StevenBock/docdiff/internal/report"
	"github.com/StevenBock/docdiff/internal/scanner"
)

type reviewBaseline struct {
	DocCommit     string
	AckRecorded   string
	AckFloor      string
	AckReanchored bool
	Effective     string
}

// computeStaleDocs returns docs whose linked files changed (in committed
// history) since the doc itself was last committed. Each doc's own last commit
// is the "reviewed" anchor — editing code and doc together in one commit makes
// them share that anchor, so nothing is stale. An `ack` floor (.docdiff-acks.json)
// can move the anchor forward for docs reviewed without an edit. Warnings go to
// errOut.
func computeStaleDocs(g *git.Git, res *scanner.Result, errOut io.Writer) map[string]*report.StaleDoc {
	filesByDoc := res.FilesByDoc
	stale := make(map[string]*report.StaleDoc)

	acks, err := loadAcks(rootDir)
	if err != nil {
		fmt.Fprintf(errOut, "Warning: failed to load %s: %v\n", acksFile, err)
		acks = map[string]string{}
	}

	structural := map[string]bool{}
	if cfg != nil {
		structural = cfg.StructuralDocs()
	}

	// Structural docs are evaluated even when no file currently links them —
	// removing a doc's last annotation IS a structural change.
	docSet := make(map[string]bool, len(filesByDoc)+len(structural))
	for doc := range filesByDoc {
		docSet[doc] = true
	}
	for doc := range structural {
		docSet[doc] = true
	}

	structuralRanges := make(map[string]*structuralRange)

	for doc := range docSet {
		files := filesByDoc[doc]
		if len(files) == 0 && !structural[doc] {
			continue
		}

		baseline, err := baselineForDoc(g, doc, acks)
		if err != nil {
			fmt.Fprintf(errOut, "Warning: failed to find last commit for %s: %v\n", doc, err)
			continue
		}

		lastHash := baseline.Effective
		if lastHash == "" {
			continue // doc not committed and not acked; nothing to compare against
		}

		var changed []string
		if structural[doc] {
			changed = structuralChangedFiles(g, doc, lastHash, files, structuralRanges, errOut)
		} else {
			changed, err = g.ChangedFilesBetween(lastHash, "HEAD", files)
			if err != nil {
				fmt.Fprintf(errOut, "Warning: failed to check changes for %s (%s..HEAD): %v\n", doc, lastHash, err)
				continue
			}
			changed = dropAnnotationOnlyChanges(g, lastHash, "HEAD", changed)
			// Match hunks against the doc's owned regions, exactly as `check`
			// does for the working tree. A scoped annotation that owns lines
			// 387-431 must not be flagged by a change at line 889.
			changed = scopedChangedFiles(g, doc, lastHash, "HEAD", changed, res.Annotations, cfg.AnnotationTag)
		}

		if len(changed) > 0 {
			commitInfo, _ := g.CommitInfo(lastHash)
			stale[doc] = &report.StaleDoc{
				Path:           doc,
				LastHash:       lastHash,
				LastCommitInfo: commitInfo,
				FilesChanged:   len(changed),
				ChangedFiles:   changed,
			}
		}
	}

	return stale
}

// structuralRange caches the whole-repo diff analysis for one baseline hash so
// several structural docs sharing an anchor don't re-run git.
type structuralRange struct {
	deltas     map[string]*fileAnnDelta
	nameStatus []git.NameStatus
}

// structuralChangedFiles applies the inverted (gate: structural) polarity over
// committed history: annotation-map changes and linked-file add/rename since
// the baseline make the doc stale; body edits never do.
func structuralChangedFiles(g *git.Git, doc, lastHash string, files []string, cache map[string]*structuralRange, errOut io.Writer) []string {
	r := cache[lastHash]
	if r == nil {
		diff, err := g.Diff(lastHash, "HEAD", nil)
		if err != nil {
			fmt.Fprintf(errOut, "Warning: failed to diff %s..HEAD for structural doc %s: %v\n", lastHash, doc, err)
			return nil
		}
		ns, err := g.NameStatusBetween(lastHash, "HEAD", nil)
		if err != nil {
			fmt.Fprintf(errOut, "Warning: failed to get name-status %s..HEAD for %s: %v\n", lastHash, doc, err)
			return nil
		}
		r = &structuralRange{deltas: annotationDeltasByFile(diff, cfg.AnnotationTag), nameStatus: ns}
		cache[lastHash] = r
	}

	linked := make(map[string]bool, len(files))
	for _, f := range files {
		linked[f] = true
	}
	trigFiles, _ := structuralTriggers(doc, r.deltas, r.nameStatus, nil, func(f string) bool { return linked[f] })
	return uniqueSorted(trigFiles)
}

// dropAnnotationOnlyChanges removes files whose entire diff in from..to is
// annotation churn — a moved or retargeted annotation line changes the
// ownership map, not the behavior a doc describes, so it must not stale it (keeps
// report/graph consistent with check, which stops gating those files too).
// Best-effort: on any git error the list passes through unfiltered.
func dropAnnotationOnlyChanges(g *git.Git, from, to string, changed []string) []string {
	if len(changed) == 0 || cfg == nil || cfg.AnnotationTag == "" {
		return changed
	}
	diff, err := g.Diff(from, to, changed)
	if err != nil {
		return changed
	}
	annOnly := annotationOnlyDiffFiles(diff, cfg.AnnotationTag)
	if len(annOnly) == 0 {
		return changed
	}
	kept := make([]string, 0, len(changed))
	for _, f := range changed {
		if !annOnly[f] {
			kept = append(kept, f)
		}
	}
	return kept
}

func baselineForDoc(g *git.Git, doc string, acks map[string]string) (reviewBaseline, error) {
	docCommit, err := g.LastCommit(doc)
	if err != nil {
		return reviewBaseline{}, err
	}

	recorded := acks[doc]
	ackFloor, reanchored := resolveAckFloor(g, doc, recorded)
	return reviewBaseline{
		DocCommit:     docCommit,
		AckRecorded:   recorded,
		AckFloor:      ackFloor,
		AckReanchored: reanchored,
		Effective:     effectiveBaseline(g, docCommit, ackFloor),
	}, nil
}

func resolveAckFloor(g *git.Git, doc, recorded string) (string, bool) {
	if recorded == "" {
		return "", false
	}

	if reachable, err := g.IsAncestor(recorded, "HEAD"); err == nil && reachable {
		return recorded, false
	}

	reanchored, err := g.LastCommitMatching(acksFile, ackEntryRegex(doc))
	if err == nil && reanchored != "" {
		return reanchored, true
	}

	return recorded, false
}

func ackEntryRegex(doc string) string {
	return regexp.QuoteMeta(strconv.Quote(doc)) + `\s*:`
}

// effectiveBaseline picks the review anchor: the newer of the doc's own last
// commit and its ack floor. A missing or unresolvable ack falls back to the
// doc's commit, so a stale/garbage-collected ack never hides real changes.
func effectiveBaseline(g *git.Git, docCommit, ackedSha string) string {
	if ackedSha == "" {
		return docCommit
	}
	if docCommit == "" {
		return ackedSha
	}
	// The ack moves the anchor forward only if the doc's commit is an ancestor
	// of it. Otherwise the doc's own commit is already at least as new.
	if anc, err := g.IsAncestor(docCommit, ackedSha); err == nil && anc {
		return ackedSha
	}
	return docCommit
}
