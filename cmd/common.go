package cmd

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"runtime"
	"strings"

	"github.com/daedaleanai/cobra"
	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/linepipes"
	"github.com/daedaleanai/reqtraq/repos"
	"github.com/daedaleanai/reqtraq/reqs"
	"github.com/daedaleanai/reqtraq/util"
	"github.com/pkg/errors"
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

// Sets up the global reqtraqConfig variable and registers the base repository
// @llr REQ-TRAQ-SWL-60
func setupConfiguration() error {
	config.LoadBaseRepoInfo()

	// Register BaseRepository so that it is always accessible afterwards
	baseRepoPath := repos.BaseRepoPath()
	repos.RegisterRepository(repos.BaseRepoName(), baseRepoPath)

	cfg, err := config.ParseConfig(baseRepoPath)
	if err != nil {
		return errors.Wrap(err, "Error parsing `reqtraq_config.json` file in current repo")
	}

	reqtraqConfig = &cfg
	return nil
}

// loadReqGraph loads the requirements graph from the current repository or
// from the specified paths of previously exported requirement graphs.
// @llr REQ-TRAQ-SWL-1, REQ-TRAQ-SWL-80
func loadReqGraph(graphs_paths []string) (*reqs.ReqGraph, error) {
	var rg *reqs.ReqGraph
	var err error
	if len(graphs_paths) == 0 {
		if err = setupConfiguration(); err != nil {
			return nil, errors.Wrap(err, "setup configuration")
		}

		rg, err = reqs.BuildGraph(reqtraqConfig)
		if err != nil {
			return nil, errors.Wrap(err, "build graph")
		}
	} else {
		rg, err = reqs.LoadGraphs(graphs_paths)
		if err != nil {
			return nil, errors.Wrap(err, "load graphs")
		}
	}
	return rg, nil
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

// Initializes the root command flags
// @llr REQ-TRAQ-SWL-32, REQ-TRAQ-SWL-59
func init() {
	rootCmd.PersistentFlags().BoolVarP(&linepipes.Verbose, "verbose", "v", false, "Enable verbose logs.")
	rootCmd.PersistentFlags().BoolVarP(&config.DirectDependenciesOnly, "direct-deps", "d", false, "Only checks the current repository and parents")
}

// Runs the root command and defers the cleanup of the temporary directories
// until it exits.
// @llr REQ-TRAQ-SWL-32, REQ-TRAQ-SWL-59
func RunRootCommand() error {
	defer repos.CleanupTemporaryDirectories()
	return rootCmd.Execute()
}

// RunAndHandleError returns a RunE function that runs the specified RunE
// function and exits if it returns an error.
// @llr REQ-TRAQ-SWL-59
func RunAndHandleError(runE func(cmd *cobra.Command, args []string) error) func(*cobra.Command, []string) error {
	// Wrap the specified runE func in a new func with the same signature.
	return func(cmd *cobra.Command, args []string) error {
		// At some place in Cobra they lose track of whether the error is
		// returned by a RunE function or it's an arguments parsing error.
		// That's why we need to handle our errors ourselves and exit with an
		// appropriate error code.
		// See https://github.com/spf13/cobra/issues/914
		if errRun := runE(cmd, args); errRun != nil {
			// For example: "github.com/daedaleanai/reqtraq/cmd.runValidate"
			s := runtime.FuncForPC(reflect.ValueOf(runE).Pointer()).Name()
			s = s[strings.LastIndex(s, "/")+1:]
			fmt.Println(errors.Wrap(errRun, s))
			os.Exit(1)
		}
		return nil
	}
}
