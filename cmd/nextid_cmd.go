package cmd

import (
	"fmt"

	"github.com/daedaleanai/cobra"
	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/repos"
	"github.com/daedaleanai/reqtraq/reqs"
)

var nextIdCmd = &cobra.Command{
	Use:               "nextid CERTDOC_PATH",
	Short:             "Generates the next requirement id for the given document",
	Long:              "Generates the next requirement id for the given document. Takes a certdoc path as a single argument",
	Args:              cobra.ExactValidArgs(1),
	ValidArgsFunction: completeCertdocFilename,
	RunE:              RunAndHandleError(runNextId),
}

// runNextId parses a single markdown document for requirements and returns the next available ID
// @llr REQ-TRAQ-SWL-34
func runNextId(command *cobra.Command, args []string) error {
	var (
		requirements  []*reqs.Req
		greatestReqID int = 0
		greatestAsmID int = 0
	)

	if err := setupConfiguration(); err != nil {
		return err
	}

	filename := args[0]

	var repoName repos.RepoName
	var certdocConfig *config.Document
	if repoName, certdocConfig = reqtraqConfig.FindCertdoc(filename); certdocConfig == nil {
		return fmt.Errorf("Could not find document `%s` in the list of documents", filename)
	}

	requirements, _, err := reqs.ParseMarkdown(repoName, certdocConfig)
	if err != nil {
		return err
	}

	// count existing REQ and ASM IDs
	for _, r := range requirements {
		if r.Variant == reqs.ReqVariantRequirement && r.IDNumber > greatestReqID {
			greatestReqID = r.IDNumber
		} else if r.Variant == reqs.ReqVariantAssumption && r.IDNumber > greatestAsmID {
			greatestAsmID = r.IDNumber
		}
	}

	fmt.Printf("REQ-%s-%s-%d\n", certdocConfig.ReqSpec.Prefix, certdocConfig.ReqSpec.Level, greatestReqID+1)

	// don't bother reporting assumptions if none are defined yet
	if greatestAsmID > 0 {
		fmt.Printf("ASM-%s-%s-%d\n", certdocConfig.ReqSpec.Prefix, certdocConfig.ReqSpec.Level, greatestAsmID+1)
	}

	return nil
}

// Registers the nexid command
// @llr REQ-TRAQ-SWL-34
func init() {
	rootCmd.AddCommand(nextIdCmd)
}
