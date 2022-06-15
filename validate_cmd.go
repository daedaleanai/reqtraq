package main

import (
	"errors"
	"fmt"

	"github.com/daedaleanai/cobra"
	"github.com/daedaleanai/reqtraq/config"
)

var fValidateStrict *bool
var fValidateAt *string

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validates the requirement documents",
	Long:  `Runs the validation checks for the requirement documents`,
	RunE:  runValidate,
}

// validate builds the requirement graph, gathering any errors and prints them out. If the strict flag is set return an error.
// @llr REQ-TRAQ-SWL-36
func validate(config *config.Config, at string, strict bool) error {
	rg, err := buildGraph(at, config)
	if err != nil {
		return err
	}

	if len(rg.Errors) > 0 {
		for _, e := range rg.Errors {
			fmt.Println(e)
		}
		if strict {
			return errors.New("ERROR. Validation failed")
		}
		fmt.Println("WARNING. Validation failed")
	} else {
		fmt.Println("Validation passed")
	}
	return nil
}

// the run command for validate
// @llr REQ-TRAQ-SWL-36
func runValidate(command *cobra.Command, args []string) error {
	if err := setupConfiguration(); err != nil {
		return err
	}
	return validate(reqtraqConfig, *fValidateAt, *fValidateStrict)
}

// Registers the validate command
// @llr REQ-TRAQ-SWL-36
func init() {
	fValidateStrict = validateCmd.PersistentFlags().Bool("strict", false, "Exit with error if any validation checks fail")
	fValidateAt = validateCmd.PersistentFlags().String("at", "", "Runs validation at the given commit instead of the current one. This only applies to the current repository")
	rootCmd.AddCommand(validateCmd)
}
