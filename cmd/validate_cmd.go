package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/daedaleanai/cobra"
	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/diagnostics"
	"github.com/daedaleanai/reqtraq/repos"
	"github.com/daedaleanai/reqtraq/reqs"
)

var fValidateStrict *bool
var fValidateJson *string
var fOnlyErrors *bool

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

// Translates the severity code into a value valid for the json output
// @llr REQ-TRAQ-SWL-66
func translateSeverityCode(severity diagnostics.IssueSeverity) string {
	switch severity {
	case diagnostics.IssueSeverityMajor:
		return "error"
	case diagnostics.IssueSeverityMinor:
		return "warning"
	case diagnostics.IssueSeverityNote:
		return "note"
	}
	return "error"
}

// Builds a Json file with the issues found after parsing the requirements and code. It only collects
// information for the base repository.
// @llr REQ-TRAQ-SWL-66
func buildJsonIssues(issues []diagnostics.Issue, jsonWriter *json.Encoder) {
	for _, issue := range issues {
		// Only report issues for the current repository
		if issue.RepoName != repos.BaseRepoName() {
			continue
		}

		var name string
		var code string
		switch issue.Type {
		case diagnostics.IssueTypeInvalidRequirementId:
			name = "Invalid requirement ID"
			code = "REQ1"
		case diagnostics.IssueTypeInvalidParent:
			name = "Invalid parent requirement"
			code = "REQ2"
		case diagnostics.IssueTypeInvalidRequirementReference:
			name = "Invalid requirement reference"
			code = "REQ3"
		case diagnostics.IssueTypeInvalidRequirementInCode:
			name = "Invalid requirement"
			code = "REQ4"
		case diagnostics.IssueTypeMissingRequirementInCode:
			name = "Code without requirements"
			code = "REQ5"
		case diagnostics.IssueTypeMissingAttribute:
			name = "Missing attribute"
			code = "REQ6"
		case diagnostics.IssueTypeUnknownAttribute:
			name = "Unknown attribute"
			code = "REQ7"
		case diagnostics.IssueTypeInvalidAttributeValue:
			name = "Invalid attribute"
			code = "REQ8"
		case diagnostics.IssueTypeReqTestedButNotImplemented:
			name = "Requirement tested but not implemented"
			code = "REQ9"
		case diagnostics.IssueTypeReqNotImplemented:
			name = "Requirement not implemented"
			code = "REQ10"
		case diagnostics.IssueTypeReqNotTested:
			name = "Requirement not tested"
			code = "REQ11"
		case diagnostics.IssueTypeNoShallInBody:
			name = "No shall statement in body"
			code = "REQ12"
		case diagnostics.IssueTypeManyShallInBody:
			name = "Multiple shall statements in body"
			code = "REQ13"
		case diagnostics.IssueTypeShallInRationale:
			name = "Shall statement in rationale attribute"
			code = "REQ14"
		default:
			log.Fatal("Unhandled IssueType: %r", issue.Type)
		}

		jsonWriter.Encode(LintMessage{
			Name:        name,
			Code:        code,
			Severity:    translateSeverityCode(issue.Severity),
			Path:        issue.Path,
			Line:        issue.Line,
			Char:        0,
			Description: issue.Error.Error(),
		})
	}
}

// validate builds the requirement graph, gathering any errors and prints them out. If the strict flag is set return an error.
// @llr REQ-TRAQ-SWL-36
func validate(config *config.Config, strict bool) ([]diagnostics.Issue, error) {
	rg, err := reqs.BuildGraph(config)
	if err != nil {
		return rg.Issues, err
	}

	hasCriticalErrors := false
	for _, issue := range rg.Issues {
		if issue.Severity != diagnostics.IssueSeverityNote {
			hasCriticalErrors = true
		} else if *fOnlyErrors {
			continue
		}
		fmt.Println(issue.Error)
	}

	if hasCriticalErrors {
		if strict {
			return rg.Issues, errors.New("ERROR. Validation failed")
		} else {
			fmt.Println("WARNING. Validation failed")
		}
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
	issues, err := validate(reqtraqConfig, *fValidateStrict)

	if *fValidateJson != "" {
		file, fileErr := os.Create(*fValidateJson)
		if fileErr != nil {
			log.Fatalf("Could not create json file %v\n", fileErr)
		}
		defer file.Close()

		jsonWriter := json.NewEncoder(file)
		buildJsonIssues(issues, jsonWriter)
	}

	if err != nil {
		log.Fatal(err)
	}

	// The return error is used when the issued command is not valid, not in the
	// case the command actually fails to run. Since no args are used by this command,
	// we can always return nil
	return nil
}

// Registers the validate command
// @llr REQ-TRAQ-SWL-36
func init() {
	fValidateStrict = validateCmd.PersistentFlags().Bool("strict", false, "Exit with error if any validation checks fail")
	fValidateJson = validateCmd.PersistentFlags().String("json", "", "Outputs a json file with lint messages in addition to a textual representation of the errors")
	fOnlyErrors = validateCmd.PersistentFlags().Bool("only-errors", false, "Only outputs actual errors")
	rootCmd.AddCommand(validateCmd)
}
