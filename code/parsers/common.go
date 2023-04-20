package parsers

import "github.com/daedaleanai/reqtraq/code"

// @llr REQ-TRAQ-SWL-8
func Register() {
	// ctags is registered explicitly. Other parsers do not need to be because they are automatically imported with
	// init functions of the parsers package.
	parser := ctagsCodeParser{}
	code.RegisterCodeParser("ctags", parser)
}
