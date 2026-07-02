package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/StevenBock/docdiff/internal/git"
)

var ackTo string

var ackCmd = &cobra.Command{
	Use:   "ack <doc>...",
	Short: "Mark a doc reviewed even though it needed no change",
	Long: `Acknowledge that a doc is up to date even though its linked code changed.

Normally a doc is marked reviewed by editing it in the same commit as its code.
When you review the code and decide the doc needs NO edit, there is nothing to
commit against it — so 'ack' records a floor commit (default HEAD) for the doc.
Staleness is then measured from the newer of the doc's own last commit and this
floor, so the doc stops being reported stale until its linked code changes again.

The floor is an existing commit, so there is no chicken-and-egg: commit your
code first, then run 'docdiff ack <doc>' and commit .docdiff-acks.json.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAck,
}

func init() {
	ackCmd.Flags().StringVar(&ackTo, "to", "", "floor commit to ack at (HEAD, branch, or sha); default HEAD")
	rootCmd.AddCommand(ackCmd)
}

func runAck(cmd *cobra.Command, args []string) error {
	g := git.New(rootDir)

	ref := ackTo
	if ref == "" {
		ref = "HEAD"
	}
	sha, err := g.ResolveShort(ref)
	if err != nil {
		return fmt.Errorf("failed to resolve %q: %w", ref, err)
	}

	acks, err := loadAcks(rootDir)
	if err != nil {
		return fmt.Errorf("failed to load acks: %w", err)
	}

	out := cmd.OutOrStdout()
	for _, doc := range args {
		doc = filepath.ToSlash(doc)
		if _, statErr := os.Stat(filepath.Join(rootDir, doc)); statErr != nil {
			return fmt.Errorf("doc not found: %s", doc)
		}
		acks[doc] = sha
		fmt.Fprintf(out, "Acked %s at %s (won't report stale until its linked code changes again)\n", doc, sha)
	}

	if err := saveAcks(rootDir, acks); err != nil {
		return fmt.Errorf("failed to save acks: %w", err)
	}

	sort.Strings(args)
	fmt.Fprintf(out, "\nCommit %s to share these acknowledgements.\n", acksFile)
	return nil
}
