package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/daedaleanai/cobra"
	"github.com/daedaleanai/reqtraq/diagnostics"
	"github.com/daedaleanai/reqtraq/repos"
	"github.com/pkg/errors"
)

var fValidateStrict *bool
var fValidateJson *string
var fPrintOnlyErrors *bool

var validateCmd = &cobra.Command{
	Use:   "validate [graph.json ...]",
	Short: "Validates the requirement documents",
	Long:  `Runs the validation checks for the requirements documents in the current repo or in the specified requirements graphs exported previously.`,
	RunE:  RunAndHandleError(runValidate),
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
func buildJsonIssues(issues []diagnostics.Issue, jsonWriter *json.Encoder) error {
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
		case diagnostics.IssueTypeInvalidFlowId:
			name = "Invalid Flow tag identifier"
			code = "REQ15"
		case diagnostics.IssueTypeFlowNotImplemented:
			name = "Flow tag is not linked to a requirement"
			code = "REQ16"
		case diagnostics.IssueTypeDuplicateFlowId:
			name = "Duplicate Flow tag identifier"
			code = "REQ17"
		case diagnostics.IssueTypeMissingFlowId:
			name = "Missing Flow tag identifier"
			code = "REQ18"
		case diagnostics.IssueTypeInvalidFlowDirection:
			name = "Invalid flow direction"
			code = "REQ19"
		case diagnostics.IssueTypeFlowIdOfDifferentItem:
			name = "Requirement references flow tag of a different item"
			code = "REQ20"
		default:
			log.Fatal("Unhandled IssueType: %r", issue.Type)
		}

		message := LintMessage{
			Name:        name,
			Code:        code,
			Severity:    translateSeverityCode(issue.Severity),
			Path:        issue.Path,
			Line:        issue.Line,
			Char:        0,
			Description: issue.Description,
		}
		if err := jsonWriter.Encode(message); err != nil {
			return err
		}
	}
	return nil
}

// createIssuesReport writes the specified requirements issues to a JSON file.
// @llr REQ-TRAQ-SWL-36
func createIssuesReport(issues []diagnostics.Issue, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	jsonWriter := json.NewEncoder(file)
	return buildJsonIssues(issues, jsonWriter)
}

// validate prints the issues detected in the requirements graph.
// Returns the count of critical issues and the count of lint messages.
// @llr REQ-TRAQ-SWL-36
func validate(issues []diagnostics.Issue, onlyErrors bool) (int, int) {
	criticalErrorsCount := 0
	lintErrorsCount := 0
	for _, issue := range issues {
		if issue.Severity == diagnostics.IssueSeverityNote {
			lintErrorsCount += 1
			if onlyErrors {
				continue
			}
		} else {
			criticalErrorsCount += 1
		}
		fmt.Println(issue.Description)
	}

	return criticalErrorsCount, lintErrorsCount
}

// the run command for validate
// @llr REQ-TRAQ-SWL-36
func runValidate(command *cobra.Command, args []string) error {
	rg, err := loadReqGraph(args)
	if err != nil {
		return errors.Wrap(err, "load req graph")
	}

	if *fValidateJson != "" {
		if err := createIssuesReport(rg.Issues, *fValidateJson); err != nil {
			return errors.Wrap(err, "create report")
		}
	}

	criticalErrorsCount, _ := validate(rg.Issues, *fPrintOnlyErrors)
	if *fValidateStrict && criticalErrorsCount > 0 {
		return fmt.Errorf("validation failed: %d critical issues", criticalErrorsCount)
	}

	fmt.Println("Validation passed!")
	return nil
}

// Registers the validate command
// @llr REQ-TRAQ-SWL-36
func init() {
	fValidateStrict = validateCmd.PersistentFlags().Bool("strict", false, "Exit with error if any validation issues are found. Only issues with severity 'minor' or 'normal' are counted, linting messages are ignored.")
	fValidateJson = validateCmd.PersistentFlags().String("json", "", "Additionally, create a JSON file with all errors and lint messages")
	fPrintOnlyErrors = validateCmd.PersistentFlags().Bool("only-errors", false, "Only output actual errors, skipping the lint messages")
	rootCmd.AddCommand(validateCmd)
}
