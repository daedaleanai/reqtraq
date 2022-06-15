package main

import (
	"fmt"
	"strings"

	"github.com/daedaleanai/cobra"
	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/repos"
)

var listCmd = &cobra.Command{
	Use:               "list [CertdocPath]",
	Short:             "Parses and lists the requirements found in certification documents",
	Long:              `Parses and lists the requirements found in certification documents. Takes a certdoc path as a single argument`,
	Args:              cobra.ExactValidArgs(1),
	ValidArgsFunction: completeCertdocFilename,
	RunE:              runListCmd,
}

// list all requirements in the given certdoc
// @llr REQ-TRAQ-SWL-33
func runListCmd(command *cobra.Command, args []string) error {
	filename := args[0]
	if err := setupConfiguration(); err != nil {
		return err
	}

	var repoName repos.RepoName
	var certdocConfig *config.Document
	if repoName, certdocConfig = reqtraqConfig.FindCertdoc(filename); certdocConfig == nil {
		return fmt.Errorf("Could not find document `%s` in the list of documents", filename)
	}
	reqs, err := ParseMarkdown(repoName, certdocConfig)
	if err != nil {
		return err
	}
	for _, r := range reqs {
		body := make([]string, 0)
		lines := strings.Split(r.Body, "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			body = append(body, line)
		}
		fmt.Printf("Requirement %s %s\n", r.ID, r.Title)
		// Check for empty body because deleted requirements have no body.
		if len(body) > 0 {
			fmt.Printf("%s…\n", body[0])
		}
		fmt.Println()
	}
	return nil
}

// Registers the list command
// @llr REQ-TRAQ-SWL-33
func init() {
	rootCmd.AddCommand(listCmd)
}
