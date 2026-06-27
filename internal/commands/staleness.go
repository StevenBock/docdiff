package commands

import (
	"fmt"
	"io"

	"github.com/StevenBock/docdiff/internal/git"
	"github.com/StevenBock/docdiff/internal/metadata"
	"github.com/StevenBock/docdiff/internal/report"
)

// computeStaleDocs returns docs whose linked files changed between their synced
// hash and HEAD (committed history). Warnings are written to errOut.
func computeStaleDocs(g *git.Git, versions metadata.DocVersions, filesByDoc map[string][]string, errOut io.Writer) map[string]*report.StaleDoc {
	stale := make(map[string]*report.StaleDoc)

	for doc, lastHash := range versions {
		files := filesByDoc[doc]
		if len(files) == 0 {
			continue
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
