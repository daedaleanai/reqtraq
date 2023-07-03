package parsers

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/daedaleanai/reqtraq/code"
	"github.com/daedaleanai/reqtraq/linepipes"
	"github.com/daedaleanai/reqtraq/repos"
	"github.com/pkg/errors"
)

type ctagsCodeParser struct{}

// TagCode runs ctags over the specified code files and parses the generated tags file.
// @llr REQ-TRAQ-SWL-8
func (ctagsCodeParser) TagCode(repoName repos.RepoName, codeFiles []code.CodeFile, compilationDatabase string, compilerArguments []string) (map[code.CodeFile][]*code.Code, error) {
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

	languages := make([]string, 0, len(code.SourceCodeFileExtensions))
	for l := range code.SourceCodeFileExtensions {
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
		"--kinds-SystemVerilog=iAVR",
		"--regex-systemverilog=/CHECK_([a-zA-Z_]+)/\\1/A/",
		"--kinds-Verilog=i",
		"--kinds-VHDL=ea",
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

	tags, err := parseTags(repoName, lines, codeFiles)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse ctags output")
	}

	if err, _ := <-errs; err != nil {
		return nil, errors.Wrap(err, "failed to run ctags to find methods in the source code")
	}

	tagsByFile := make(map[code.CodeFile][]*code.Code, 0)
	for _, tag := range tags {
		_, ok := tagsByFile[tag.CodeFile]
		if !ok {
			tagsByFile[tag.CodeFile] = make([]*code.Code, 0)
		}
		tagsByFile[tag.CodeFile] = append(tagsByFile[tag.CodeFile], tag)
	}
	return tagsByFile, nil
}

// parseTags takes the raw output from Universal Ctags and parses into Code structs.
// @llr REQ-TRAQ-SWL-8
func parseTags(repoName repos.RepoName, lines chan string, codeFiles []code.CodeFile) ([]*code.Code, error) {
	codeFilesMap := map[string]code.CodeFile{}
	for _, codeFile := range codeFiles {
		codeFilesMap[codeFile.Path] = codeFile
	}

	res := make([]*code.Code, 0)
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
		res = append(res, &code.Code{CodeFile: codeFilesMap[relativePath], Tag: tag, Line: line})
	}
	return res, nil
}

const installUniversalCtags = "Make sure to install Universal ctags (NOT Exuberant ctags) as described in https://github.com/universal-ctags/ctags#the-latest-build-and-package"

// checkCtagsAvailable returns an error when Universal Ctags cannot be found.
// @llr REQ-TRAQ-SWL-8
func checkCtagsAvailable() error {
	out, err := linepipes.All(linepipes.Run(findCtags(), "--version"))
	if err != nil {
		return errors.Wrap(err, "universal-ctags not available. "+installUniversalCtags)
	}
	if !strings.Contains(out, "Universal Ctags") {
		return fmt.Errorf("`ctags` tool is not universal-ctags. " + installUniversalCtags)
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
	for _, extensions := range code.SourceCodeFileExtensions {
		for _, e := range extensions {
			if e == ext {
				return true
			}
		}
	}
	return false
}
