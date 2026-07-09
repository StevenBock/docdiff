package commands

// @doc CLAUDE.md

import (
	"fmt"
	"io"
	"regexp"
	"strconv"

	"github.com/StevenBock/docdiff/internal/git"
	"github.com/StevenBock/docdiff/internal/report"
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
func computeStaleDocs(g *git.Git, filesByDoc map[string][]string, errOut io.Writer) map[string]*report.StaleDoc {
	stale := make(map[string]*report.StaleDoc)

	acks, err := loadAcks(rootDir)
	if err != nil {
		fmt.Fprintf(errOut, "Warning: failed to load %s: %v\n", acksFile, err)
		acks = map[string]string{}
	}

	for doc, files := range filesByDoc {
		if len(files) == 0 {
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

		changed, err := g.ChangedFilesBetween(lastHash, "HEAD", files)
		if err != nil {
			fmt.Fprintf(errOut, "Warning: failed to check changes for %s (%s..HEAD): %v\n", doc, lastHash, err)
			continue
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
