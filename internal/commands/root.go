package commands

import (
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"

	"github.com/StevenBock/docdiff/internal/config"
	"github.com/StevenBock/docdiff/internal/language"
)

var (
	rootDir  string
	cfg      *config.Config
	registry *language.Registry
)

var rootCmd = &cobra.Command{
	Use:   "docdiff",
	Short: "Detect stale documentation across your codebase",
	Long: `docdiff tracks which source files relate to which documentation files
and detects when code changes require documentation updates.

Use @doc annotations in source file comments to link code to docs.
Example: a comment containing "@doc docs/API.md"`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error

		if rootDir == "" {
			rootDir, err = os.Getwd()
			if err != nil {
				return err
			}
		}

		cfg, err = config.Load(rootDir)
		if err != nil {
			return err
		}

		registry = language.DefaultRegistry()

		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&rootDir, "dir", "", "project root directory (default: current directory)")
}

func Execute() error {
	return rootCmd.Execute()
}

func GetConfig() *config.Config {
	return cfg
}

func GetRegistry() *language.Registry {
	return registry
}

func GetRootDir() string {
	return rootDir
}

// buildRevision is the commit this binary was built from, read from Go's VCS
// build stamps. Deliberately not a hand-maintained version constant: a string
// nobody remembers to bump reads as authoritative while being wrong, which is
// worse than absent. Empty when built without VCS info (-buildvcs=false, or
// from the module cache).
func buildRevision() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	var rev, modified string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			modified = s.Value
		}
	}
	if rev != "" && modified == "true" {
		rev += "+dirty" // built from uncommitted changes; the sha alone would lie
	}
	return rev
}
