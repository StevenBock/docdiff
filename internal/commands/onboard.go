package commands

import (
	_ "embed"
	"fmt"

	"github.com/spf13/cobra"
)

//go:embed skills/docdiff/SKILL.md
var docdiffSkillContent string

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
	out := cmd.OutOrStdout()
	if _, err := fmt.Fprint(out, onboardText); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out, onboardSkillText, docdiffSkillContent)
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

A doc is "reviewed" as of its own last commit. If a linked source file has a
newer commit than the doc, the doc is stale — so editing code and doc together
in one commit keeps it fresh. There is no metadata file and nothing to sync.

## Key Commands

- ` + "`docdiff check`" + `      — Show ONLY docs affected by your current changes (--staged, --files, --json). Exits non-zero if an affected doc needs updating. Best command for focused agent work.
- ` + "`docdiff report`" + `     — Full repo-wide stale/orphaned report (supports --json, --sarif, --ci)
- ` + "`docdiff changes <doc>`" + ` — Show code changes since a doc was last committed (--ai, --working-tree, --staged)
- ` + "`docdiff ack <doc>`" + `   — Mark a doc reviewed when its code changed but the doc needed NO edit (records a floor commit in .docdiff-acks.json; --to <ref>)
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
2. Develop normally — edit code
3. Run ` + "`docdiff check`" + ` to see which affected docs still need updating
4. Run ` + "`docdiff changes <doc> --working-tree`" + ` to see what changed
5. Update the documentation
6. Commit the code and doc together — the shared commit marks the doc reviewed

---

## Agent Instructions Snippet

The block below is ready to paste into your agent instruction file.
Target files: CLAUDE.md, .github/copilot-instructions.md, .cursorrules, .windsurfrules, AGENTS.md

--- START DOCDIFF INSTRUCTIONS ---

## docdiff — Keeping Documentation In Sync

This project uses docdiff to track documentation freshness. Source files contain
` + "`@doc`" + ` annotations (e.g. ` + "`// @doc docs/API.md`" + `) that link code to documentation.

### When modifying code with @doc annotations:
- Run ` + "`docdiff check`" + ` to see ONLY the docs affected by your current (uncommitted) changes, ignoring unrelated stale docs elsewhere in the repo. Exit code is non-zero while an affected doc still needs updating.
- Run ` + "`docdiff changes <doc> --working-tree`" + ` to see exactly what changed (including uncommitted edits) in files linked to that doc.
- Update the linked documentation to reflect your code changes.
- Commit the code and doc TOGETHER in one commit. A doc is reviewed as of its own last commit, so sharing the commit with its linked code marks it fresh — no separate sync step and no second commit.
- If you review a doc and it genuinely needs NO change, run ` + "`docdiff ack <doc>`" + ` after committing the code and commit ` + "`.docdiff-acks.json`" + `. This records a floor so the doc stops reporting stale until its linked code changes again.

### When adding new code:
- Add ` + "`@doc`" + ` annotations linking to the relevant documentation file
- Supported comment styles: ` + "`//`" + `, ` + "`#`" + `, ` + "`/* */`" + `, ` + "`/** */`" + `, ` + "`<!-- -->`" + `

### When adding new documentation:
- Add ` + "`@doc`" + ` annotations in relevant source files pointing to the new doc
- Commit the doc and those source files together

--- END DOCDIFF INSTRUCTIONS ---
`

const onboardSkillText = `
---

## Optional Coding-Agent Skill

If you are a coding agent reading this output, ask the user before installing
the bundled docdiff skill. Do not write any skill files unless the user says yes.

Suggested prompt:

    Install the docdiff skill for coding agents into .claude/skills/docdiff/SKILL.md? [y/N]

If the user agrees, create .claude/skills/docdiff/SKILL.md (or the equivalent
skills directory for this agent harness, such as
.agents/skills/docdiff/SKILL.md) with this exact content:

~~~markdown
%s
~~~
`
