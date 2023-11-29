package cmd

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/daedaleanai/cobra"
	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/repos"
	"github.com/daedaleanai/reqtraq/reqs"
)

var (
	listIdFilter        *string
	listTitleFilter     *string
	listBodyFilter      *string
	listAttributeFilter *[]string

	listCsvFormat *bool
)

var listCmd = &cobra.Command{
	Use:               "list CERTDOC_PATH",
	Short:             "Parses and lists the requirements found in a certification document",
	Long:              `Parses and lists the requirements found in a certification document. Takes a certdoc path as a single argument`,
	Args:              cobra.ExactValidArgs(1),
	ValidArgsFunction: completeCertdocFilename,
	RunE:              RunAndHandleError(runListCmd),
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
	requirements, _, err := reqs.ParseMarkdown(repoName, certdocConfig)
	if err != nil {
		return err
	}
	filter, err := reqs.CreateFilter(*listIdFilter, *listTitleFilter, *listBodyFilter, *listAttributeFilter)
	if err != nil {
		return err
	}
	if *listCsvFormat {
		printCsv(requirements, certdocConfig.Schema, filter)
	} else {
		printConcise(requirements, filter)
	}
	return nil
}

// printConcise prints to stdout the requirements which match the filter in a concise format (id, title and first line of body text)
// @llr REQ-TRAQ-SWL-33, REQ-TRAQ-SWL-73
func printConcise(reqs []*reqs.Req, filter reqs.ReqFilter) {
	for _, r := range reqs {
		if !filter.IsEmpty() && !r.Matches(&filter) {
			continue
		}
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
}

// printCsv prints to stdout the requirements which match the filter in csv format, including all attributes
// @llr REQ-TRAQ-SWL-33, REQ-TRAQ-SWL-73, REQ-TRAQ-SWL-74
func printCsv(reqs []*reqs.Req, schema config.Schema, filter reqs.ReqFilter) {
	csvwriter := csv.NewWriter(os.Stdout)
	var attributes []string
	for a := range schema.Attributes {
		attributes = append(attributes, cases.Title(language.BritishEnglish).String(a))
	}
	sort.Strings(attributes)
	csvwriter.Write(append([]string{"Id", "Title", "Body"}, attributes...))
	for _, r := range reqs {
		if !filter.IsEmpty() && !r.Matches(&filter) {
			continue
		}
		row := []string{r.ID, r.Title, r.Body}
		for _, a := range attributes {
			row = append(row, r.Attributes[strings.ToUpper(a)])
		}
		csvwriter.Write(row)
	}
	csvwriter.Flush()
}

// Registers the list command
// @llr REQ-TRAQ-SWL-33
func init() {
	listIdFilter = listCmd.PersistentFlags().String("id", "", "Regular expression to filter by requirement id.")
	listTitleFilter = listCmd.PersistentFlags().String("title", "", "Regular expression to filter by requirement title.")
	listBodyFilter = listCmd.PersistentFlags().String("body", "", "Regular expression to filter by requirement body.")
	listAttributeFilter = listCmd.PersistentFlags().StringSlice("attribute", nil, "Regular expression to filter by requirement attribute.")

	listCsvFormat = listCmd.PersistentFlags().Bool("csv", false, "Output in csv format.")

	rootCmd.AddCommand(listCmd)
}
