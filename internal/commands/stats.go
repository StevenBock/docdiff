package commands

// @doc README.md

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/StevenBock/docdiff/internal/git"
	"github.com/StevenBock/docdiff/internal/scanner"
)

var statsJSON bool

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Per-doc ack-vs-edit history ratios",
	Long: `Show, per doc, how often its linked code was acked (reviewed with no
edit) versus how often the doc itself was edited, over the whole history.

A doc that is mostly acked and rarely edited tracks code it doesn't actually
describe — a candidate for a 'gate: structural' policy (see the docs: config
map) or for deletion. Read-only report; changes nothing.`,
	RunE: runStats,
}

func init() {
	statsCmd.Flags().BoolVar(&statsJSON, "json", false, "output as JSON")
	rootCmd.AddCommand(statsCmd)
}

type docStats struct {
	Doc   string  `json:"doc"`
	Edits int     `json:"edits"`
	Acks  int     `json:"acks"`
	Ratio float64 `json:"ratio"` // acks per edit; edits==0 reported as acks
}

func runStats(cmd *cobra.Command, args []string) error {
	g := git.New(rootDir)

	s := scanner.New(cfg, registry)
	scanResult, err := s.Scan(rootDir)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	ackCounts := ackCountsFromHistory(g)

	docs := make(map[string]bool, len(scanResult.FilesByDoc)+len(ackCounts))
	for doc := range scanResult.FilesByDoc {
		docs[doc] = true
	}
	for doc := range ackCounts {
		docs[doc] = true
	}

	rows := make([]docStats, 0, len(docs))
	for doc := range docs {
		edits, err := g.CommitCount(doc)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to count commits for %s: %v\n", doc, err)
			continue
		}
		acks := ackCounts[doc]
		ratio := float64(acks)
		if edits > 0 {
			ratio = float64(acks) / float64(edits)
		}
		rows = append(rows, docStats{Doc: doc, Edits: edits, Acks: acks, Ratio: ratio})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Ratio != rows[j].Ratio {
			return rows[i].Ratio > rows[j].Ratio
		}
		return rows[i].Doc < rows[j].Doc
	})

	out := cmd.OutOrStdout()
	if statsJSON {
		data, _ := json.MarshalIndent(struct {
			Docs []docStats `json:"docs"`
		}{rows}, "", "  ")
		fmt.Fprintln(out, string(data))
		return nil
	}

	if len(rows) == 0 {
		fmt.Fprintln(out, "No docs found.")
		return nil
	}
	fmt.Fprintln(out, "Ack-vs-edit history per doc (high ratio = reviewed by ack, rarely edited):")
	fmt.Fprintf(out, "  %5s  %5s  %6s  %s\n", "ACKS", "EDITS", "RATIO", "DOC")
	for _, r := range rows {
		hint := ""
		if r.Acks >= 2 && r.Acks > r.Edits {
			hint = "  <- consider 'gate: structural' or deleting the doc"
		}
		fmt.Fprintf(out, "  %5d  %5d  %6.1f  %s%s\n", r.Acks, r.Edits, r.Ratio, r.Doc, hint)
	}
	return nil
}

// ackCountsFromHistory replays .docdiff-acks.json across its git history and
// counts, per doc, how many times its floor changed (first appearance counts).
func ackCountsFromHistory(g *git.Git) map[string]int {
	counts := make(map[string]int)
	history, err := g.FileHistory(acksFile)
	if err != nil {
		return counts
	}
	prev := map[string]string{}
	for _, sha := range history {
		content, err := g.ShowFileAt(sha, acksFile)
		if err != nil {
			continue // file absent at this commit (e.g. deleted later)
		}
		entries, err := parseAckEntries([]byte(content))
		if err != nil {
			continue
		}
		cur := make(map[string]string, len(entries))
		for doc, e := range entries {
			cur[doc] = e.Commit
		}
		for doc, floor := range cur {
			if prev[doc] != floor {
				counts[doc]++
			}
		}
		prev = cur
	}
	return counts
}
