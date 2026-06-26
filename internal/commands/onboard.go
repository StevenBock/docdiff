package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var onboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Print docdiff usage instructions for AI agents",
	Long: `Print comprehensive docdiff usage instructions to stdout.

Designed for AI agents to read and incorporate into their workflow.
Includes a ready-to-paste snippet for agent instruction files
(CLAUDE.md, .github/copilot-instructions.md, .cursorrules, etc).

This command works without any project setup — no config file or
git repository is needed.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	RunE: runOnboard,
}

func init() {
	rootCmd.AddCommand(onboardCmd)
}

func runOnboard(cmd *cobra.Command, args []string) error {
	_, err := fmt.Fprint(cmd.OutOrStdout(), onboardText)
	return err
}

const onboardText = `# docdiff — Stale Documentation Detector

docdiff is a CLI tool that detects stale documentation by tracking ` + "`@doc`" + ` annotations
in source code comments. When code changes but linked documentation hasn't been updated,
docdiff flags it as stale.

## How @doc Annotations Work

Add ` + "`@doc`" + ` annotations in source code comments to link code to documentation files:

    // @doc docs/API.md
    func HandleRequest() { ... }

    # @doc docs/GUIDE.md
    def process_data():
        ...

    /** @doc docs/ARCHITECTURE.md */
    public class Router { ... }

When the annotated code changes but the linked doc hasn't been reviewed, docdiff
reports it as stale.

## Key Commands

- ` + "`docdiff init`" + `       — Initialize documentation version tracking (creates metadata file)
- ` + "`docdiff report`" + `     — Show stale and orphaned docs (supports --json, --sarif, --ci)
- ` + "`docdiff changes <doc>`" + ` — Show code changes since a doc was last updated (--ai for LLM-friendly output)
- ` + "`docdiff sync [doc]`" + `  — Update metadata after reviewing/updating docs
- ` + "`docdiff graph`" + `      — Output doc-to-file relationship graph (DOT or --mermaid)
- ` + "`docdiff onboard`" + `    — Print these instructions

## Configuration

docdiff reads ` + "`.docdiff.yaml`" + ` or ` + "`.docdiff.json`" + ` from the project root. Key settings:

- ` + "`annotation_tag`" + `    — Customize the tag (default: ` + "`@doc`" + `)
- ` + "`include/exclude`" + `   — Glob patterns for file scanning
- ` + "`docs_directory`" + `    — Where documentation lives (default: ` + "`docs`" + `)
- ` + "`ci.fail_on_stale`" + `  — Control exit code behavior in CI

## Supported Languages

Go, Rust, Java, PHP, Python, JavaScript/TypeScript, Ruby, Vue, Shell/Bash, and PowerShell.

## Typical Workflow

1. Add ` + "`@doc`" + ` annotations to source files linking them to documentation
2. Run ` + "`docdiff init`" + ` to create the metadata baseline
3. Develop normally — edit code and documentation
4. Run ` + "`docdiff report`" + ` to see which docs are stale
5. Run ` + "`docdiff changes <doc>`" + ` to see what changed
6. Update the documentation
7. Run ` + "`docdiff sync`" + ` to mark docs as reviewed

---

## Agent Instructions Snippet

The block below is ready to paste into your agent instruction file.
Target files: CLAUDE.md, .github/copilot-instructions.md, .cursorrules, .windsurfrules, AGENTS.md

--- START DOCDIFF INSTRUCTIONS ---

## docdiff — Keeping Documentation In Sync

This project uses docdiff to track documentation freshness. Source files contain
` + "`@doc`" + ` annotations (e.g. ` + "`// @doc docs/API.md`" + `) that link code to documentation.

### When modifying code with @doc annotations:
- Run ` + "`docdiff report`" + ` to check for stale docs
- Run ` + "`docdiff changes <doc>`" + ` to see what code changed since the doc was last reviewed
- Update the linked documentation to reflect your code changes
- Run ` + "`docdiff sync`" + ` to mark the documentation as up-to-date

### When adding new code:
- Add ` + "`@doc`" + ` annotations linking to the relevant documentation file
- Supported comment styles: ` + "`//`" + `, ` + "`#`" + `, ` + "`/* */`" + `, ` + "`/** */`" + `, ` + "`<!-- -->`" + `

### When adding new documentation:
- Run ` + "`docdiff sync <doc>`" + ` to register the new doc in metadata
- Add ` + "`@doc`" + ` annotations in relevant source files

--- END DOCDIFF INSTRUCTIONS ---
`
