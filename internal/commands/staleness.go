package commands

import (
	"fmt"
	"io"

	"github.com/StevenBock/docdiff/internal/git"
	"github.com/StevenBock/docdiff/internal/report"
)

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

		lastHash, err := g.LastCommit(doc)
		if err != nil {
			fmt.Fprintf(errOut, "Warning: failed to find last commit for %s: %v\n", doc, err)
			continue
		}

		lastHash = effectiveBaseline(g, lastHash, acks[doc])
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
