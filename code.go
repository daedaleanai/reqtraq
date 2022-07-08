/*
Functions which deal with source code files. Source code is discovered within a given path and searched for functions and associated requirement IDs. The external program Universal Ctags is used to scan for functions.
*/

package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/linepipes"
	"github.com/daedaleanai/reqtraq/repos"
	"github.com/pkg/errors"
)

var (
	// To detect a line containing low-level requirements
	reLLRReferenceLine = regexp.MustCompile(`^[ \*\/]*(?:@|\\)llr +(?:REQ-\w+-\w+-\d+[, ]*)+$`)
	// To capture requirements out of the line
	reLLRReferences = regexp.MustCompile(`(REQ-\w+-\w+-\d+)`)
	// Blank line to stop search
	reBlankLine = regexp.MustCompile(`^\s*$`)
	// List of supported code parsers. ctags is always built-in. Other parsers will be registered
	// during runtime by calling registerCodeParser
	codeParsers = map[string]CodeParser{"ctags": CtagsCodeParser{}}
)

// An interface for a code parser.
type CodeParser interface {
	tagCode(repoName repos.RepoName,
		codeFiles []CodeFile,
		compilationDatabase string,
		CompilerArguments []string) (map[CodeFile][]*Code, error)
}

// Registers a code parser with the given name
// @llr REQ-TRAQ-SWL-65
func registerCodeParser(name string, codeParser CodeParser) {
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
}

// Returns a string with the name of the repository and the path in it where the code file can be found
// @llr REQ-TRAQ-SWL-38
func (codeFile *CodeFile) String() string {
	return fmt.Sprintf("%s: %s", codeFile.RepoName, codeFile.Path)
}

// Code represents a code node in the graph of requirements.
type Code struct {
	// The file where the code can be found
	CodeFile CodeFile
	// Tag is the name of the function.
	Tag string
	// Line number where the function starts.
	Line int
	// Requirement IDs found in the comment above the function.
	ParentIds []string
	Parents   []*Req
	// Link back to its parent document. Used to validate the requirements belong to this document
	Document *config.Document
}

// ParseCode is the entry point for the code related functions. It parses all tags found in the
// implementation for the given document. The return value is a map from each discovered source code
// file to a slice of Code structs representing the functions found within.
// @llr REQ-TRAQ-SWL-8 REQ-TRAQ-SWL-9, REQ-TRAQ-SWL-61
func ParseCode(repoName repos.RepoName, document *config.Document) (map[CodeFile][]*Code, error) {
	// Create a list with all the files to parse
	codeFiles := make([]CodeFile, 0)
	codeFilePaths := make([]string, 0)
	for _, codeFile := range document.Implementation.CodeFiles {
		codeFiles = append(codeFiles, CodeFile{
			RepoName: repoName,
			Path:     codeFile,
		})
		codeFilePaths = append(codeFilePaths, codeFile)
	}
	for _, testFile := range document.Implementation.TestFiles {
		codeFiles = append(codeFiles, CodeFile{
			RepoName: repoName,
			Path:     testFile,
		})
		codeFilePaths = append(codeFilePaths, testFile)
	}

	// TODO(ja): Distinguish between code and tests so that we can get the test coverage and the
	// source coverage separately

	if len(codeFiles) == 0 {
		// In order to avoid calling tagCode and having the default ctags parser
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

	tags, err = codeParser.tagCode(repoName, codeFiles,
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
	"C":   {".c", ".h"},
	"C++": {".cc", ".hh"},
	"GO":  {".go"},
}

const InstallUniversalCtags = "Make sure to install Universal ctags (NOT Exuberant ctags) as described in https://github.com/universal-ctags/ctags#the-latest-build-and-package"

// checkCtagsAvailable returns an error when Universal Ctags cannot be found.
// @llr REQ-TRAQ-SWL-8
func checkCtagsAvailable() error {
	out, err := linepipes.All(linepipes.Run(findCtags(), "--version"))
	if err != nil {
		return errors.Wrap(err, "universal-ctags not available. "+InstallUniversalCtags)
	}
	if !strings.Contains(out, "Universal Ctags") {
		return fmt.Errorf("`ctags` tool is not universal-ctags. " + InstallUniversalCtags)
	}
	return nil
}

// findCtags returns the location of the Universal Ctags executable.
// @llr REQ-TRAQ-SWL-8
func findCtags() string {
	ctags, ok := os.LookupEnv("REQTRAQ_CTAGS")
	if !ok {
		ctags = "ctags"
	}
	return ctags
}

// isSourceCodeFile returns true if the provided file name has the extension of a supported source code type
// @llr REQ-TRAQ-SWL-6
func isSourceCodeFile(name string) bool {
	ext := strings.ToLower(path.Ext(name))
	for _, extensions := range SourceCodeFileExtensions {
		for _, e := range extensions {
			if e == ext {
				return true
			}
		}
	}
	return false
}

// parseComments updates the specified tags with the requirement IDs discovered in the codeFiles.
// @llr REQ-TRAQ-SWL-9
func parseComments(codeTags map[CodeFile][]*Code) error {
	for codeFile := range codeTags {
		fsPath, err := repos.PathInRepo(codeFile.RepoName, codeFile.Path)
		if err != nil {
			return err
		}
		if err := parseFileComments(fsPath, codeTags[codeFile]); err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed comments discovery for %s - %s", codeFile.RepoName, codeFile.Path))
		}
	}
	return nil
}

// parseFileComments detects comments in the specified source code file, parses them for requirements IDs and
// associates them with the tags detected in the same file.
// @llr REQ-TRAQ-SWL-9
func parseFileComments(absolutePath string, tags []*Code) error {
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
		if tags[i].Line == previousTag {
			// If there's a duplicate tag then just copy the links and continue
			tags[i].ParentIds = tags[i-1].ParentIds
			continue
		}
		tags[i].ParentIds = []string{}
		for lineNo := tags[i].Line - 1; lineNo > previousTag; lineNo-- {
			if reLLRReferenceLine.MatchString(sourceLines[lineNo]) {
				// Looks good, extract all references straight into the tag
				tags[i].ParentIds = append(tags[i].ParentIds, reLLRReferences.FindAllString(sourceLines[lineNo], -1)...)
			} else if reBlankLine.MatchString(sourceLines[lineNo]) {
				// We've hit a blank line
				break
			}

		}
		previousTag = tags[i].Line
	}

	return nil
}

// parseTags takes the raw output from Universal Ctags and parses into Code structs.
// @llr REQ-TRAQ-SWL-8
func parseTags(repoName repos.RepoName, lines chan string) ([]*Code, error) {
	res := make([]*Code, 0)
	for line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) < 4 {
			// Most probably some lines with metadata info about
			// the ctags generator.
			continue
		}
		tag := parts[0]
		if strings.HasPrefix(tag, "__anon") {
			// Ignore anonymous functions like lambdas
			continue
		}
		p := parts[1]
		if !isSourceCodeFile(p) {
			continue
		}
		repoPath, err := repos.GetRepoPathByName(repoName)
		if err != nil {
			return nil, err
		}
		relativePath, err := filepath.Rel(string(repoPath), p)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to parse path on line: %v", parts))
		}
		if !strings.HasPrefix(parts[3], "line:") {
			return nil, fmt.Errorf("line number unknown prefix: %v", parts)
		}
		line, err := strconv.Atoi(parts[3][5:])
		if err != nil {
			return nil, fmt.Errorf("failed to parse line number: %v", parts)
		}
		res = append(res, &Code{CodeFile: CodeFile{RepoName: repoName, Path: relativePath}, Tag: tag, Line: line})
	}
	return res, nil
}

type CtagsCodeParser struct{}

// tagCode runs ctags over the specified code files and parses the generated tags file.
// @llr REQ-TRAQ-SWL-8
func (CtagsCodeParser) tagCode(repoName repos.RepoName, codeFiles []CodeFile, compilationDatabase string, compilerArguments []string) (map[CodeFile][]*Code, error) {
	r, w := io.Pipe()
	errChannel := make(chan error)
	go func(errChannel chan error) {
		for _, codeFile := range codeFiles {
			codePath, err := repos.PathInRepo(repoName, codeFile.Path)
			if err != nil {
				errChannel <- err
				return
			}
			fmt.Fprintln(w, codePath)
		}
		w.Close()
	}(errChannel)

	languages := make([]string, 0, len(SourceCodeFileExtensions))
	for l := range SourceCodeFileExtensions {
		languages = append(languages, l)
	}

	if err := checkCtagsAvailable(); err != nil {
		return nil, errors.Wrap(err, "need to use Universal ctags to tag the code")
	}
	lines, errs := linepipes.RunWithInput(findCtags(), r,
		// We're only interested in a limited set of languages.
		// Avoid scanning JSON, Markdown, etc.
		"--languages="+strings.Join(languages, ","),
		// To see the available kinds for a language: ctags --list-kinds-full=C++
		// We're interested only in functions.
		"--kinds-C=f",
		"--kinds-C++=f",
		"--kinds-GO=f",
		// To see the available fields: ctags --list-fields
		// We need the line number.
		"--fields=n",
		"--recurse",
		"-f", "-",
		"-L", "-")

	select {
	case err := <-errChannel:
		return nil, err
	default:
	}

	tags, err := parseTags(repoName, lines)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse ctags output")
	}

	if err, _ := <-errs; err != nil {
		return nil, errors.Wrap(err, "failed to run ctags to find methods in the source code")
	}

	tagsByFile := make(map[CodeFile][]*Code, 0)
	for _, tag := range tags {
		_, ok := tagsByFile[tag.CodeFile]
		if !ok {
			tagsByFile[tag.CodeFile] = make([]*Code, 0)
		}
		tagsByFile[tag.CodeFile] = append(tagsByFile[tag.CodeFile], tag)
	}
	return tagsByFile, nil
}
