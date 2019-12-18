package main

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/daedaleanai/reqtraq/git"
	"github.com/daedaleanai/reqtraq/linepipes"
	"github.com/pkg/errors"
)

// SourceCodeFileExtensions are listed by language.
// To see the available languages, run: ctags --list-languages
var SourceCodeFileExtensions = map[string][]string{
	"C":   {".c", ".h"},
	"C++": {".cc", ".hh"},
	"GO":  {".go"},
}

const InstallUniversalCtags = "Install as described in https://github.com/universal-ctags/ctags#the-latest-build-and-package"

// runCtags executes Universal Ctags with the specified arguments.
func runCtags(args ...string) (lines chan string, errors chan error) {
	ctags, ok := os.LookupEnv("REQTRAQ_CTAGS")
	if !ok {
		ctags = "ctags"
	}
	return linepipes.Run(ctags, args...)
}

// checkCtagsAvailable returns an error when Universal Ctags cannot be found.
func checkCtagsAvailable() error {
	out, err := linepipes.All(runCtags("--version"))
	if err != nil {
		return errors.Wrap(err, "universal-ctags not available. "+InstallUniversalCtags)
	}
	if !strings.Contains(out, "Universal Ctags") {
		return fmt.Errorf("`ctags` tool is not universal-ctags. " + InstallUniversalCtags)
	}
	return nil
}

// parseTags returns the tags found by ctags.
func parseTags(lines chan string) ([]Code, error) {
	res := make([]Code, 0)
	for line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) < 4 {
			// Most probably some lines with metadata info about
			// the ctags generator.
			continue
		}
		tag := parts[0]
		p := parts[1]
		if !IsSourceCodeFile(p) {
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
		res = append(res, Code{Path: relativePath, Tag: tag, Line: line})
	}
	return res, nil
}

// parseFileComments detects comments in the specified source code file and
// associates them with the tags detected in the same file.
// Returns a hash for the file
func parseFileComments(absolutePath string, tags []Code) (string, error) {
	f, err := os.Open(absolutePath)
	if err != nil {
		return "", err
	}
	h := sha1.New()
	// git compatible hash
	if s, err := f.Stat(); err == nil {
		fmt.Fprintf(h, "blob %d", s.Size())
		h.Write([]byte{0})
	}

	// Detect comments.
	comments := make(map[int][]string, 0)
	scanner := bufio.NewScanner(io.TeeReader(f, h))
	var comment []string
	var lineNumber int
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "*/") || strings.HasPrefix(line, "/*") || strings.HasPrefix(line, "* ") {
			if comment == nil {
				comment = make([]string, 0)
			}
			comment = append(comment, line)
		} else {
			if comment != nil {
				// This is the first line after the comment.
				comments[lineNumber] = comment
				comment = nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", errors.Wrap(err, "failed to read source code file")
	}

	// Associate the detected comments with the tags.
	for i := range tags {
		if comment, ok := comments[tags[i].Line]; ok {
			tags[i].Comment = comment
		}
	}

	return string(h.Sum(nil)), nil
}

// tagCode runs ctags over the specified directory and parses the generated
// tags file.
func tagCode(codePath string) (map[string][]Code, error) {
	absoluteCodePath := filepath.Join(git.RepoPath(), codePath)
	languages := make([]string, 0, len(SourceCodeFileExtensions))
	for l := range SourceCodeFileExtensions {
		languages = append(languages, l)
	}
	lines, errs := runCtags(
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
		"--recurse", "-f", "-", absoluteCodePath)

	tags, err := parseTags(lines)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse ctags output")
	}

	if err, _ := <-errs; err != nil {
		return nil, errors.Wrap(err, "failed to run ctags to find methods in the source code")
	}

	tagsByFile := make(map[string][]Code, 0)
	for _, tag := range tags {
		_, ok := tagsByFile[tag.Path]
		if !ok {
			tagsByFile[tag.Path] = make([]Code, 0)
		}
		tagsByFile[tag.Path] = append(tagsByFile[tag.Path], tag)
	}
	return tagsByFile, nil
}

func IsSourceCodeDir(dirPath string) bool {
	return path.Base(dirPath) != "testdata"
}

func IsSourceCodeFile(name string) bool {
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

// parseComments updates the specified tags with the discovered code comments.
func (rg *reqGraph) parseComments(absoluteCodePath string) error {
	// Walk through the code.
	err := filepath.Walk(absoluteCodePath, func(filePath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			if !IsSourceCodeDir(filePath) {
				return filepath.SkipDir
			}
		} else {
			if !IsSourceCodeFile(info.Name()) {
				return nil
			}

			p, err := relativePathToRepo(filePath, git.RepoPath())
			if err != nil {

			}
			if _, err := parseFileComments(filePath, rg.CodeTags[p]); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}
