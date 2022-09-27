/*
 * Reqtraq is the swiss army knife binary implementing all requirements tracking and linting for prod repo's at Daedalean.
 * Run without arguments to get comprehensive help.
 */

package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/daedaleanai/cobra"
	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/linepipes"
	"github.com/daedaleanai/reqtraq/repos"
	"github.com/daedaleanai/reqtraq/util"
)

var rootCmd = &cobra.Command{
	Use:   "reqtraq",
	Short: "Reqtraq is a requirements tracer.",
	Long: `Reqtraq operates on certification documents and source code in a directory tree,
usually in a git repo.  The certification documents are scanned for requirements,
and the source code for references to them.`,
	Version: fmt.Sprintf("%d.%d.%d", util.Version.Major, util.Version.Minor, util.Version.Revision),
}

var reqtraqConfig *config.Config

// Runs the root command and defers the cleanup of the temporary directories until it exits
// @llr REQ-TRAQ-SWL-59
func Execute() error {
	defer repos.CleanupTemporaryDirectories()

	rootCmd.PersistentFlags().BoolVarP(&linepipes.Verbose, "verbose", "v", false, "Enable verbose logs.")
	rootCmd.PersistentFlags().BoolVarP(&config.DirectDependenciesOnly, "direct-deps", "d", false, "Only checks the current repository and parents")

	return rootCmd.Execute()
}

// Runs the program
// @llr REQ-TRAQ-SWL-59
func main() {
	if Execute() != nil {
		os.Exit(1)
	}
}

// Sets up the global reqtraqConfig variable and registers the base repository
// @llr REQ-TRAQ-SWL-60
func setupConfiguration() error {
	config.LoadBaseRepoInfo()

	// Register BaseRepository so that it is always accessible afterwards
	baseRepoPath := repos.BaseRepoPath()
	repos.RegisterRepository(repos.BaseRepoName(), baseRepoPath)

	cfg, err := config.ParseConfig(baseRepoPath)
	if err != nil {
		return fmt.Errorf("Error parsing `reqtraq_config.json` file in current repo: %v", err)
	}

	reqtraqConfig = &cfg
	return nil
}

// Provides completions for certdocs
// @llr REQ-TRAQ-SWL-57
func completeCertdocFilename(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	possibleCompletions := []string{}
	if len(args) >= 1 {
		return possibleCompletions, cobra.ShellCompDirectiveNoFileComp
	}
	if err := setupConfiguration(); err != nil {
		log.Fatalf("Unable to get completions: %s", err.Error())
	}
	for repoName := range reqtraqConfig.Repos {
		for docIdx := range reqtraqConfig.Repos[repoName].Documents {
			docPath := reqtraqConfig.Repos[repoName].Documents[docIdx].Path
			if strings.HasPrefix(docPath, toComplete) {
				possibleCompletions = append(possibleCompletions, docPath)
			}
		}
	}
	return possibleCompletions, cobra.ShellCompDirectiveDefault
}
