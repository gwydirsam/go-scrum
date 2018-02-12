package main

import (
	"os"

	"github.com/gwydirsam/go-scrum/cmd/scrum/cmd"
	"github.com/gwydirsam/go-scrum/cmd/scrum/internal/buildtime"
	"github.com/rs/zerolog/log"
	"github.com/sean-/sysexits"
)

var (
	// Variables populated by govvv(1).
	Version    = "dev"
	BuildDate  string
	GitCommit  string
	GitBranch  string
	GitState   string
	GitSummary string
)

func realMain() int {
	exportBuildtimeConsts()

	if err := cmd.Execute(); err != nil {
		log.Error().Err(err).Msg("unable to run")
		return 1
	}

	return sysexits.OK
}

func main() {
	os.Exit(realMain())
}

func exportBuildtimeConsts() {
	buildtime.GitCommit = GitCommit
	buildtime.GitBranch = GitBranch
	buildtime.GitState = GitState
	buildtime.GitSummary = GitSummary
	buildtime.BuildDate = BuildDate
	buildtime.Version = Version
}
