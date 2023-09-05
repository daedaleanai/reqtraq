/*
 * Reqtraq is the swiss army knife binary implementing all requirements tracking and linting for prod repo's at Daedalean.
 * Run without arguments to get comprehensive help.
 */

package main

import (
	"os"

	"github.com/daedaleanai/reqtraq/cmd"
	"github.com/daedaleanai/reqtraq/code/parsers"
)

// @llr REQ-TRAQ-SWL-59
func init() {
	parsers.Register()
}

// Runs the program
// @llr REQ-TRAQ-SWL-59
func main() {
	if cmd.RunRootCommand() != nil {
		// Errors that get to this point should be arguments parsing errors.
		os.Exit(2)
	}
}
