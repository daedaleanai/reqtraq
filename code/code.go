/*
Functions which deal with source code files. Source code is discovered within a given path and searched for functions and associated requirement IDs. The external program Universal Ctags is used to scan for functions.
*/

package code

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/repos"
	"github.com/pkg/errors"
)

var (
	// To detect a line containing low-level requirements
	reLLRReferenceLine = regexp.MustCompile(`^[ \*\/-]*(?:@|\\)llr +(?:REQ-\w+-\w+-\d+[, ]*)+$`)
	// To capture requirements out of the line
	reLLRReferences = regexp.MustCompile(`(REQ-\w+-\w+-\d+)`)
	// Blank line to stop search
	reBlankLine = regexp.MustCompile(`^\s*$`)
	// List of supported code parsers. ctags is always built-in. Other parsers will be registered
	// during runtime by calling registerCodeParser
	codeParsers = map[string]CodeParser{}
)

// An interface for a code parser.
type CodeParser interface {
	TagCode(repoName repos.RepoName,
		codeFiles []CodeFile,
		compilationDatabase string,
		CompilerArguments []string) (map[CodeFile][]*Code, error)
}

// The type of code
type CodeType uint

const (
	CodeTypeImplementation CodeType = iota
	CodeTypeTests
	CodeTypeAny
)

// @llr REQ-TRAQ-SWL-70
func (codeType CodeType) String() string {
	switch codeType {
	case CodeTypeImplementation:
		return "Implementation"
	case CodeTypeTests:
		return "Tests"
	case CodeTypeAny:
		return "Implementation and tests"
	}
	log.Fatal("Unknown code type!")
	return "Unknown"
}

// @llr REQ-TRAQ-SWL-70
func (codeType CodeType) Matches(requested CodeType) bool {
	return (requested == CodeTypeAny) || (codeType == requested)
}

// Registers a code parser with the given name
// @llr REQ-TRAQ-SWL-65
func RegisterCodeParser(name string, codeParser CodeParser) {
	codeParsers[name] = codeParser
}

// Lists all available code parsers by name (key)
// @llr REQ-TRAQ-SWL-65
func availableCodeParsers() []string {
	list := []string{}
	for name := range codeParsers {
		list = append(list, name)
	}
	return list
}

type CodeFile struct {
	RepoName repos.RepoName
	Path     string
	Type     CodeType
}

// Returns a string with the name of the repository and the path in it where the code file can be found
// @llr REQ-TRAQ-SWL-38
func (codeFile *CodeFile) String() string {
	return fmt.Sprintf("%s: %s", codeFile.RepoName, codeFile.Path)
}

type Position struct {
	Line      uint
	Character uint
}

type Range struct {
	Start Position
	End   Position
}

type ReqLink struct {
	Id    string
	Range Range
}

// Code represents a code node in the graph of requirements.
type Code struct {
	// The file where the code can be found
	CodeFile CodeFile
	// Tag is the name of the function.
	Tag string
	// Unique symbol identifier
	Symbol string
	// Line number where the function starts.
	Line int
	// Requirement IDs found in the comment above the function.
	Links []ReqLink
	// Link back to its parent document. Used to validate the requirements belong to this document
	Document *config.Document
	// Whether the code MUST link to a requirement or simply CAN link to a requirement
	Optional bool
}

// byFilenameTag provides sort functions to order code by their repo name, then path value, and then line number
type byFilenameTag []*Code

// @llr REQ-TRAQ-SWL-47
func (a byFilenameTag) Len() int { return len(a) }

// @llr REQ-TRAQ-SWL-47
func (a byFilenameTag) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// @llr REQ-TRAQ-SWL-47
func (a byFilenameTag) Less(i, j int) bool {
	switch strings.Compare(string(a[i].CodeFile.RepoName), string(a[j].CodeFile.RepoName)) {
	case -1:
		return true
	case 0:
		switch strings.Compare(a[i].CodeFile.Path, a[j].CodeFile.Path) {
		case -1:
			return true
		case 0:
			return a[i].Line < a[j].Line
		}
		return false
	}
	return false
}

// ParseCode is the entry point for the code related functions. It parses all tags found in the
// implementation for the given document. The return value is a map from each discovered source code
// file to a slice of Code structs representing the functions found within.
// @llr REQ-TRAQ-SWL-8 REQ-TRAQ-SWL-9, REQ-TRAQ-SWL-61, REQ-TRAQ-SWL-69
func ParseCode(repoName repos.RepoName, document *config.Document) (map[CodeFile][]*Code, error) {
	// Create a list with all the files to parse
	codeFiles := make([]CodeFile, 0)
	codeFilePaths := make([]string, 0)
	for _, implFile := range document.Implementation.CodeFiles {
		codeFiles = append(codeFiles, CodeFile{
			RepoName: repoName,
			Path:     implFile,
			Type:     CodeTypeImplementation,
		})
		codeFilePaths = append(codeFilePaths, implFile)
	}
	for _, testFile := range document.Implementation.TestFiles {
		codeFiles = append(codeFiles, CodeFile{
			RepoName: repoName,
			Path:     testFile,
			Type:     CodeTypeTests,
		})
		codeFilePaths = append(codeFilePaths, testFile)
	}

	if len(codeFiles) == 0 {
		// In order to avoid calling TagCode and having the default ctags parser
		// check that ctags is installed we can simply return here.
		// That way, those users that don't need ctags don't have to install it.
		return make(map[CodeFile][]*Code), nil
	}

	var tags map[CodeFile][]*Code
	var err error

	codeParser, ok := codeParsers[document.Implementation.CodeParser]
	if !ok {
		return nil, fmt.Errorf("Code parser not found: `%s`\n\tAvailable parsers: %v", document.Implementation.CodeParser, availableCodeParsers())
	}

	tags, err = codeParser.TagCode(repoName, codeFiles,
		document.Implementation.CompilationDatabase, document.Implementation.CompilerArguments)
	if err != nil {
		return nil, errors.Wrap(err, "failed to tag code")
	}

	// Annotate the code procedures with the associated requirement IDs.
	if err := parseComments(tags); err != nil {
		return tags, errors.Wrap(err, "failed walking code")
	}

	for codeFile := range tags {
		for tagIdx := range tags[codeFile] {
			tags[codeFile][tagIdx].Document = document
		}
	}

	return tags, nil
}

// Create a URL path to a code function by concatenating the repository name, the source code path
// and line number of the function
// @llr REQ-TRAQ-SWL-38
func (code *Code) URL() string {
	return fmt.Sprintf("/code/%s/%s#L%d", code.CodeFile.RepoName, code.CodeFile.Path, code.Line)
}

// SourceCodeFileExtensions are listed by language.
// To see the available languages, run: ctags --list-languages
var SourceCodeFileExtensions = map[string][]string{
	"C":             {".c", ".h"},
	"C++":           {".cc", ".hh"},
	"GO":            {".go"},
	"SystemVerilog": {".sv", ".svh"},
	"Verilog":       {".v", ".vh"},
	"VHDL":          {".vhd", ".vhdl"},
}

// parseComments updates the specified tags with the requirement IDs discovered in the codeFiles.
// @llr REQ-TRAQ-SWL-9, REQ-TRAQ-SWL-75
func parseComments(codeTags map[CodeFile][]*Code) error {
	for codeFile := range codeTags {
		fsPath, err := repos.PathInRepo(codeFile.RepoName, codeFile.Path)
		if err != nil {
			return err
		}
		isTestFile := codeFile.Type.Matches(CodeTypeTests)
		if err := parseFileComments(fsPath, codeTags[codeFile], isTestFile); err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed comments discovery for %s - %s", codeFile.RepoName, codeFile.Path))
		}
	}
	return nil
}

// parseFileComments detects comments in the specified source code file, parses them for requirements IDs and
// associates them with the tags detected in the same file.
// @llr REQ-TRAQ-SWL-9, REQ-TRAQ-SWL-75
func parseFileComments(absolutePath string, tags []*Code, isTestFile bool) error {
	// Read in the source code and break into string slice
	sourceRaw, err := os.ReadFile(absolutePath)
	if err != nil {
		return err
	}
	sourceLines := strings.Split(string(sourceRaw), "\n")

	// Sort the tags so they're in line number order
	sort.Sort(byFilenameTag(tags))

	// For each tag, search through the source code backwards looking for requirement references
	previousTag := 0
	for i := range tags {
		if isTestFile {
			// Test code can link to requirements but does not need to. In principle, only testcases should be linked.
			tags[i].Optional = true
		}
		if tags[i].Line == previousTag {
			// If there's a duplicate tag then just copy the links and continue
			tags[i].Links = tags[i-1].Links
			continue
		}
		tags[i].Links = []ReqLink{}
		for lineNo := tags[i].Line - 1; lineNo > previousTag; lineNo-- {
			if reLLRReferenceLine.MatchString(sourceLines[lineNo]) {
				// Looks good, extract all references straight into the tag
				matches := reLLRReferences.FindAllStringIndex(sourceLines[lineNo], -1)
				for _, match := range matches {
					link := ReqLink{
						Id: sourceLines[lineNo][match[0]:match[1]],
						Range: Range{
							Start: Position{Line: uint(lineNo), Character: uint(match[0])},
							End:   Position{Line: uint(lineNo), Character: uint(match[1])},
						},
					}
					tags[i].Links = append(tags[i].Links, link)
				}
			} else if reBlankLine.MatchString(sourceLines[lineNo]) {
				// We've hit a blank line
				break
			}

		}
		previousTag = tags[i].Line
	}

	return nil
}
