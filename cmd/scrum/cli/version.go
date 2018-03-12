package cli

import (
	"fmt"

	"github.com/gwydirsam/go-scrum/cmd/scrum/internal/buildtime"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:          "version",
	Short:        "Display " + buildtime.PROGNAME + " version and build information",
	SilenceUsage: true,

	PreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},

	RunE: func(cmd *cobra.Command, args []string) error {
		log.Debug().
			Str("commit", buildtime.GitCommit).
			Str("branch", buildtime.GitBranch).
			Str("state", buildtime.GitState).
			Str("summary", buildtime.GitSummary).
			Str("build-date", buildtime.BuildDate).
			Str("version", buildtime.Version).
			Msg("version")

		fmt.Printf("Version: %s\n", buildtime.Version)

		{
			commit := buildtime.GitCommit
			if buildtime.GitState != "clean" {
				commit += "+" + buildtime.GitState
			}

			fmt.Printf("Commit: %s\n", commit)
		}

		fmt.Printf("Build Date: %s\n", buildtime.BuildDate)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
