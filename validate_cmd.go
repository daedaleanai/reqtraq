package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/daedaleanai/cobra"
	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/repos"
)

var fValidateStrict *bool
var fValidateAt *string
var fValidateJson *string

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validates the requirement documents",
	Long:  `Runs the validation checks for the requirement documents`,
	RunE:  runValidate,
}

type LintMessage struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Severity    string `json:"severity"`
	Path        string `json:"path"`
	Line        int    `json:"line"`
	Char        int    `json:"char"`
	Description string `json:"description"`
}

// Builds a Json file with the issues found after parsing the requirements and code. It only collects
// information for the base repository.
// @llr REQ-TRAQ-SWL-66
func buildJsonIssues(issues []Issue, jsonWriter *json.Encoder) {
	for _, issue := range issues {
		// Only report issues for the current repository
		if issue.RepoName != repos.BaseRepoName() {
			continue
		}

		var name string
		var code string
		switch issue.Type {
		case IssueTypeInvalidRequirementId:
			name = "Invalid requirement ID"
			code = "REQ1"
		case IssueTypeInvalidParent:
			name = "Invalid parent requirement"
			code = "REQ2"
		case IssueTypeInvalidRequirementReference:
			name = "Invalid requirement reference"
			code = "REQ3"
		case IssueTypeInvalidRequirementInCode:
			name = "Invalid requirement"
			code = "REQ4"
		case IssueTypeMissingRequirementInCode:
			name = "Code without requirements"
			code = "REQ5"
		case IssueTypeMissingAttribute:
			name = "Missing attribute"
			code = "REQ6"
		case IssueTypeUnknownAttribute:
			name = "Unknown attribute"
			code = "REQ7"
		case IssueTypeInvalidAttributeValue:
			name = "Invalid attribute"
			code = "REQ8"
		}

		jsonWriter.Encode(LintMessage{
			Name:        name,
			Code:        code,
			Severity:    "error",
			Path:        issue.Path,
			Line:        issue.Line,
			Char:        0,
			Description: issue.Error.Error(),
		})
	}
}

// validate builds the requirement graph, gathering any errors and prints them out. If the strict flag is set return an error.
// @llr REQ-TRAQ-SWL-36
func validate(config *config.Config, at string, strict bool) ([]Issue, error) {
	rg, err := buildGraph(at, config)
	if err != nil {
		return rg.Issues, err
	}

	if len(rg.Issues) > 0 {
		for _, issue := range rg.Issues {
			fmt.Println(issue.Error)
		}
		if strict {
			return rg.Issues, errors.New("ERROR. Validation failed")
		}
		fmt.Println("WARNING. Validation failed")
	} else {
		fmt.Println("Validation passed")
	}

	return rg.Issues, nil
}

// the run command for validate
// @llr REQ-TRAQ-SWL-36
func runValidate(command *cobra.Command, args []string) error {
	if err := setupConfiguration(); err != nil {
		return err
	}
	issues, err := validate(reqtraqConfig, *fValidateAt, *fValidateStrict)

	if *fValidateJson != "" {
		file, fileErr := os.Create(*fValidateJson)
		if fileErr != nil {
			log.Fatalf("Could not create json file %v\n", fileErr)
		}
		defer file.Close()

		jsonWriter := json.NewEncoder(file)
		buildJsonIssues(issues, jsonWriter)
	}

	return err
}

// Registers the validate command
// @llr REQ-TRAQ-SWL-36
func init() {
	fValidateStrict = validateCmd.PersistentFlags().Bool("strict", false, "Exit with error if any validation checks fail")
	fValidateAt = validateCmd.PersistentFlags().String("at", "", "Runs validation at the given commit instead of the current one. This only applies to the current repository")
	fValidateJson = validateCmd.PersistentFlags().String("json", "", "Outputs a json file with lint messages in addition to a textual representation of the errors")
	rootCmd.AddCommand(validateCmd)
}
