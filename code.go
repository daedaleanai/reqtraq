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

	"github.com/daedaleanai/reqtraq/git"
	"github.com/daedaleanai/reqtraq/linepipes"
	"github.com/pkg/errors"
)

var (
	// To detect a line containing low-level requirements
	reLLRReferenceLine = regexp.MustCompile(`^[ \*\/]*(?:@|\\)llr (?:REQ-\w+-\w+-\d+[, ]*)+$`)
	// To capture requirements out of the line
	reLLRReferences = regexp.MustCompile(`(REQ-\w+-\w+-\d+)`)
	// Blank line to stop search
	reBlankLine = regexp.MustCompile(`^\s*$`)
)

// Code represents a code node in the graph of requirements.
type Code struct {
	// Path is the code file this was found in relative to repo root.
	Path string
	// Tag is the name of the function.
	Tag string
	// Line number where the function starts.
	Line int
	// Requirement IDs found in the comment above the function.
	ParentIds []string
	Parents   []*Req
}

// ParseCode is the entry point for the code related functions. Given a path containing source code the code files are found, the procedures within them, along with their associated requirement IDs, discovered. The return value is a map from each discovered source code file to a slice of Code structs representing the functions found within.
// @llr REQ-TRAQ-SWL-6 REQ-TRAQ-SWL-8 REQ-TRAQ-SWL-9
func ParseCode(codePath string) (map[string][]*Code, error) {

	var codeFiles []string
	var codeTags map[string][]*Code
	var err error

	// Find the code files.
	codeFiles, err = findCodeFiles(codePath)
	if err != nil {
		return codeTags, errors.Wrap(err, "failed to find code files")
	}

	// Discover the code procedures.
	codeTags, err = tagCode(codeFiles)
	if err != nil {
		return codeTags, errors.Wrap(err, "failed to tag code")
	}

	// Annotate the code procedures with the associated requirement IDs.
	if err := parseComments(codeFiles, codeTags); err != nil {
		return codeTags, errors.Wrap(err, "failed walking code")
	}

	return codeTags, nil
}

// URL create a URL path to a code function by concatenating the source code path and line number of the function
// @llr REQ-TRAQ-SWL-38
func (code *Code) URL() string {
	return fmt.Sprintf("/code/%s#L%d", code.Path, code.Line)
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

// findCodeFiles returns the paths to the discovered code files in codePath relative to current git repository path.
// @llr REQ-TRAQ-SWL-6 REQ-TRAQ-SWL-7
func findCodeFiles(codePath string) ([]string, error) {
	repoPath := git.RepoPath()
	files := make([]string, 0)
	absoluteCodePath := filepath.Join(repoPath, codePath)
	err := filepath.Walk(absoluteCodePath, func(filePath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			if !isSourceCodeDir(filePath) {
				return filepath.SkipDir
			}
		} else {
			if !isSourceCodeFile(info.Name()) {
				return nil
			}
			p, err := relativePathToRepo(filePath, repoPath)
			if err != nil {
				return err
			}
			files = append(files, p)
		}
		return nil
	})
	return files, err
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

// isSourceCodeDir returns true if the provided path name should be scanned for source code files
// @llr REQ-TRAQ-SWL-7
func isSourceCodeDir(dirPath string) bool {
	return path.Base(dirPath) != "testdata"
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
func parseComments(codeFiles []string, codeTags map[string][]*Code) error {
	repoPath := git.RepoPath()
	for _, filePath := range codeFiles {
		absoluteFilePath := path.Join(repoPath, filePath)
		if err := parseFileComments(absoluteFilePath, codeTags[filePath]); err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed comments discovery for %s", absoluteFilePath))
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
		for lineNo := tags[i].Line - 1; lineNo > previousTag; lineNo-- {
			if reLLRReferenceLine.MatchString(sourceLines[lineNo]) {
				// Looks good, extract all references straight into the tag
				tags[i].ParentIds = reLLRReferences.FindAllString(sourceLines[lineNo], -1)
				break
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
func parseTags(lines chan string) ([]*Code, error) {
	res := make([]*Code, 0)
	for line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) < 4 {
			// Most probably some lines with metadata info about
			// the ctags generator.
			continue
		}
		tag := parts[0]
		p := parts[1]
		if !isSourceCodeFile(p) {
			continue
		}
		relativePath, err := relativePathToRepo(p, git.RepoPath())
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
		res = append(res, &Code{Path: relativePath, Tag: tag, Line: line})
	}
	return res, nil
}

// relativePathToRepo returns filePath relative to repoPath by
// removing the path to the repository from filePath
// @llr REQ-TRAQ-SWL-6
func relativePathToRepo(filePath, repoPath string) (string, error) {
	if filePath[:1] != "/" {
		// Already a relative path.
		return filePath, nil
	}
	fields := strings.SplitN(filePath, repoPath, 2)
	if len(fields) < 2 {
		return "", fmt.Errorf("malformed code file path: %s not in %s", filePath, repoPath)
	}
	res := fields[1]
	if res[:1] == "/" {
		res = res[1:]
	}
	return res, nil
}

// tagCode runs ctags over the specified code files and parses the generated tags file.
// @llr REQ-TRAQ-SWL-8
func tagCode(codeFiles []string) (map[string][]*Code, error) {
	repoPath := git.RepoPath()
	r, w := io.Pipe()
	go func() {
		for _, f := range codeFiles {
			fmt.Fprintln(w, path.Join(repoPath, f))
		}
		w.Close()
	}()

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

	tags, err := parseTags(lines)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse ctags output")
	}

	if err, _ := <-errs; err != nil {
		return nil, errors.Wrap(err, "failed to run ctags to find methods in the source code")
	}

	tagsByFile := make(map[string][]*Code, 0)
	for _, tag := range tags {
		_, ok := tagsByFile[tag.Path]
		if !ok {
			tagsByFile[tag.Path] = make([]*Code, 0)
		}
		tagsByFile[tag.Path] = append(tagsByFile[tag.Path], tag)
	}
	return tagsByFile, nil
}
