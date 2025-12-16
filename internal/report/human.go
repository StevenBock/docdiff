package report

import (
	"bytes"
	"fmt"
	"sort"
)

type HumanFormatter struct {
	ShowStaleOnly    bool
	ShowOrphanedOnly bool
	Tag              string
}

func (h *HumanFormatter) Format(report *Report) ([]byte, error) {
	if h.ShowStaleOnly {
		return h.formatStaleOnly(report)
	}

	if h.ShowOrphanedOnly {
		return h.formatOrphanedOnly(report)
	}

	return h.formatFull(report)
}

func (h *HumanFormatter) formatFull(report *Report) ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteString("Documentation Coverage Report\n")
	buf.WriteString("=============================\n\n")

	if len(report.StaleDocs) > 0 {
		buf.WriteString("STALE DOCS (code changed since doc updated):\n")
		staleDocs := sortedKeys(report.StaleDocs)
		for _, path := range staleDocs {
			stale := report.StaleDocs[path]
			fmt.Fprintf(&buf, "  ! %s\n", path)
			fmt.Fprintf(&buf, "    Last updated: %s\n", stale.LastCommitInfo)
			fmt.Fprintf(&buf, "    Files changed since: %d\n", stale.FilesChanged)
			fmt.Fprintf(&buf, "    Run: docdiff changes %s\n\n", path)
		}
	} else {
		buf.WriteString("No stale docs found. All documentation is up to date.\n\n")
	}

	buf.WriteString("By Documentation File:\n")
	docs := report.Metadata.SortedDocs()
	for _, doc := range docs {
		files := report.FilesByDoc[doc]
		staleMarker := ""
		if _, ok := report.StaleDocs[doc]; ok {
			staleMarker = " (stale)"
		}
		fmt.Fprintf(&buf, "  %s (%d files)%s\n", doc, len(files), staleMarker)

		shown := min(5, len(files))
		for _, file := range files[:shown] {
			fmt.Fprintf(&buf, "    %s\n", file)
		}
		if len(files) > 5 {
			fmt.Fprintf(&buf, "    ... and %d more\n", len(files)-5)
		}
	}
	buf.WriteString("\n")

	if len(report.OrphanedFiles) > 0 {
		tag := h.Tag
		if tag == "" {
			tag = "@doc"
		}
		fmt.Fprintf(&buf, "Orphaned Files (no %s): %d\n", tag, len(report.OrphanedFiles))
		shown := min(10, len(report.OrphanedFiles))
		for _, file := range report.OrphanedFiles[:shown] {
			fmt.Fprintf(&buf, "  %s\n", file)
		}
		if len(report.OrphanedFiles) > 10 {
			fmt.Fprintf(&buf, "  ... and %d more\n", len(report.OrphanedFiles)-10)
		}
		buf.WriteString("\n")
	}

	buf.WriteString("Summary:\n")
	fmt.Fprintf(&buf, "  Documented: %d/%d files (%.1f%%)\n",
		report.Summary.DocumentedFiles,
		report.Summary.TotalFiles,
		report.Summary.CoveragePercent)
	fmt.Fprintf(&buf, "  Stale docs: %d/%d\n", report.Summary.StaleDocs, report.Summary.TotalDocs)

	return buf.Bytes(), nil
}

func (h *HumanFormatter) formatStaleOnly(report *Report) ([]byte, error) {
	var buf bytes.Buffer

	if len(report.StaleDocs) == 0 {
		buf.WriteString("No stale docs found. All documentation is up to date.\n")
		return buf.Bytes(), nil
	}

	buf.WriteString("STALE DOCS (code changed since doc updated):\n\n")
	staleDocs := sortedKeys(report.StaleDocs)
	for _, path := range staleDocs {
		stale := report.StaleDocs[path]
		fmt.Fprintf(&buf, "  ! %s\n", path)
		fmt.Fprintf(&buf, "    Last updated: %s\n", stale.LastCommitInfo)
		fmt.Fprintf(&buf, "    Files changed since: %d\n", stale.FilesChanged)
		fmt.Fprintf(&buf, "    Run: docdiff changes %s\n\n", path)
	}

	return buf.Bytes(), nil
}

func (h *HumanFormatter) formatOrphanedOnly(report *Report) ([]byte, error) {
	var buf bytes.Buffer

	if len(report.OrphanedFiles) == 0 {
		tag := h.Tag
		if tag == "" {
			tag = "@doc"
		}
		fmt.Fprintf(&buf, "No orphaned files. All source files have %s annotations.\n", tag)
		return buf.Bytes(), nil
	}

	tag := h.Tag
	if tag == "" {
		tag = "@doc"
	}
	fmt.Fprintf(&buf, "Orphaned Files (no %s): %d\n\n", tag, len(report.OrphanedFiles))
	for _, file := range report.OrphanedFiles {
		fmt.Fprintf(&buf, "  %s\n", file)
	}

	return buf.Bytes(), nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
