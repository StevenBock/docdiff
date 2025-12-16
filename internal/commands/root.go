package commands

import (
	"os"

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
Example: // @doc docs/API.md`,
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
