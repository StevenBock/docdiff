package commands

// @doc README.md

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/StevenBock/docdiff/internal/git"
	"github.com/StevenBock/docdiff/internal/scanner"
)

var explainCmd = &cobra.Command{
	Use:   "explain <doc>",
	Short: "Explain why a doc is (or isn't) stale",
	Long: `Show the full staleness reasoning for a single doc in one place: its
linked files, its review anchor (last commit), any ack floor, the effective
baseline, the newest linked commit, and whether uncommitted working-tree
changes contribute — so you don't have to run several 'changes' commands.`,
	Args: cobra.ExactArgs(1),
	RunE: runExplain,
}

func init() {
	rootCmd.AddCommand(explainCmd)
}

func runExplain(cmd *cobra.Command, args []string) error {
	doc := args[0]
	out := cmd.OutOrStdout()

	s := scanner.New(cfg, registry)
	scanResult, err := s.Scan(rootDir)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	files := scanResult.FilesByDoc[doc]
	sort.Strings(files)

	fmt.Fprintf(out, "Doc: %s\n", doc)
	if len(files) == 0 {
		fmt.Fprintf(out, "\nNo source files link to this doc with %s annotations — nothing to be stale against.\n", cfg.AnnotationTag)
		return nil
	}

	fmt.Fprintf(out, "\nLinked files (%d):\n", len(files))
	for _, f := range files {
		fmt.Fprintf(out, "  %s\n", f)
	}

	g := git.New(rootDir)

	acks, err := loadAcks(rootDir)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to load %s: %v\n", acksFile, err)
		acks = map[string]string{}
	}
	baselineInfo, err := baselineForDoc(g, doc, acks)
	if err != nil {
		return fmt.Errorf("failed to find last commit for %s: %w", doc, err)
	}
	docCommit := baselineInfo.DocCommit
	ackFloor := baselineInfo.AckFloor
	baseline := baselineInfo.Effective

	fmt.Fprintln(out, "\nBaseline (review anchor):")
	if docCommit == "" {
		fmt.Fprintln(out, "  doc last commit: (uncommitted — no anchor)")
	} else {
		info, _ := g.CommitInfo(docCommit)
		fmt.Fprintf(out, "  doc last commit: %s\n", info)
	}
	if ackFloor == "" {
		fmt.Fprintln(out, "  ack floor:       none")
	} else {
		info, _ := g.CommitInfo(ackFloor)
		if baselineInfo.AckReanchored {
			// "amended" named only one cause. A squashed lane landing rewrites
			// the floor the same way and is the more common trigger in practice.
			fmt.Fprintf(out, "  ack floor:       %s (re-anchored; recorded floor %s was rewritten by an amend or squash)\n", info, baselineInfo.AckRecorded)
		} else {
			fmt.Fprintf(out, "  ack floor:       %s\n", info)
		}
		// The floor says a human signed this off; the reason says why it was
		// safe, which is the part that decays without a record.
		if entries, err := loadAckEntries(rootDir); err == nil && entries[doc].Reason != "" {
			fmt.Fprintf(out, "  ack reason:      %s\n", entries[doc].Reason)
		}
	}
	if baseline == "" {
		fmt.Fprintln(out, "  effective:       none — cannot compute staleness (doc never committed, no ack)")
		return nil
	}
	if baseline == ackFloor && baseline != docCommit {
		fmt.Fprintf(out, "  effective:       %s (from ack floor)\n", baseline)
	} else {
		fmt.Fprintf(out, "  effective:       %s\n", baseline)
	}

	// Committed drift: linked files with commits after the baseline, narrowed to
	// the ones whose hunks land in a region this doc owns. Without the narrowing
	// explain names a commit and file with no signal that the change sits 450
	// lines outside the doc's scope — and it's the one place someone goes to
	// learn WHY a doc is stale.
	// committedAll is every linked file that changed; committed is the subset
	// this doc actually owns. The working-tree section subtracts the FORMER, or
	// a file whose only change is committed-but-out-of-scope would resurface
	// below as though it were uncommitted.
	committedAll, _ := g.ChangedFilesBetween(baseline, "HEAD", files)
	committed := dropAnnotationOnlyChanges(g, baseline, "HEAD", committedAll)
	committed = scopedChangedFiles(g, doc, baseline, "HEAD", committed, scanResult.Annotations, cfg.AnnotationTag)
	commits, _ := g.CommitsBetween(baseline, "HEAD", committed)

	fmt.Fprintln(out, "\nSince baseline:")
	if len(committed) == 0 {
		fmt.Fprintln(out, "  committed changes: none")
	} else {
		newest := ""
		if len(commits) > 0 {
			newest = commits[0] // git log is newest-first
		}
		fmt.Fprintf(out, "  committed changes: %d file(s) over %d commit(s)\n", len(committed), len(commits))
		if newest != "" {
			fmt.Fprintf(out, "  newest linked commit: %s\n", newest)
		}
		for _, f := range committed {
			fmt.Fprintf(out, "    %s\n", f)
		}
	}

	// Working-tree contribution: uncommitted changes to linked files on top of
	// what's already committed since the baseline.
	sinceWorkTree, _ := g.ChangedFilesSince(baseline, false, files)
	committedSet := map[string]bool{}
	for _, f := range committedAll {
		committedSet[f] = true
	}
	var uncommitted []string
	for _, f := range sinceWorkTree {
		if !committedSet[f] {
			uncommitted = append(uncommitted, f)
		}
	}
	sort.Strings(uncommitted)
	if len(uncommitted) > 0 {
		fmt.Fprintf(out, "  working-tree changes: %d uncommitted file(s)\n", len(uncommitted))
		for _, f := range uncommitted {
			fmt.Fprintf(out, "    %s\n", f)
		}
	} else {
		fmt.Fprintln(out, "  working-tree changes: none")
	}

	fmt.Fprintln(out)
	// A `gate: structural` doc is omitted from report's stale list however much
	// body-edit drift it has, so an unqualified STALE here contradicts what the
	// user just saw in report. Say which polarity is in force.
	structuralNote := ""
	if cfg.StructuralDocs()[doc] {
		structuralNote = " (gate: structural — NOT reported stale on this; only annotation-map changes and linked-file add/delete/rename gate this doc)"
	}
	switch {
	case len(committed) > 0:
		fmt.Fprintf(out, "Verdict: STALE%s — linked code changed since the doc's baseline. Run 'docdiff changes %s' for the diff.\n", structuralNote, doc)
	case len(uncommitted) > 0:
		fmt.Fprintf(out, "Verdict: not stale in committed history, but uncommitted work touches linked files (run 'docdiff check').\n")
	default:
		fmt.Fprintln(out, "Verdict: up to date — no linked code changed since the baseline.")
	}
	return nil
}
