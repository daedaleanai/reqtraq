//go:build clang

/*
Parses the libclang AST and collects functions which are a target for requirements tracking.
*/

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/daedaleanai/reqtraq/repos"
	"github.com/go-clang/clang-v14/clang"
)

// Finds a matching compile command for a file in the repository
// @llr REQ-TRAQ-SWL-61
func findMatchingCommand(pathInRepo string, commands clang.CompileCommands) *clang.CompileCommand {
	for i := uint32(0); i < commands.Size(); i++ {
		command := commands.Command(i)
		if filepath.IsAbs(command.Filename()) {
			if command.Filename() == pathInRepo {
				return &command
			}
		} else {
			absPath, err := filepath.Abs(filepath.Join(command.Directory(), command.Filename()))
			if err != nil {
				return nil
			}
			if absPath == pathInRepo {
				return &command
			}
		}
	}
	return nil
}

// Returns true if the cursor is pointing to a possibly public entity
// @llr REQ-TRAQ-SWL-63
func IsPublic(cursor clang.Cursor) bool {
	if cursor.IsNull() {
		return true
	}

	if cursor.AccessSpecifier() != clang.AccessSpecifier_Public && cursor.AccessSpecifier() != clang.AccessSpecifier_Invalid {
		return false
	}

	return IsPublic(cursor.SemanticParent())
}

// Returns true if the cursor is part of an anonymous (or detail) namespace or class
// @llr REQ-TRAQ-SWL-62
func IsInAnonymousOrDetailNamespaceOrClass(cursor clang.Cursor) bool {
	if cursor.IsNull() {
		return false
	}

	if cursor.Kind() == clang.Cursor_Namespace && (cursor.Spelling() == "" || cursor.Spelling() == "detail") {
		return true
	}

	if ((cursor.Kind() == clang.Cursor_ClassDecl) ||
		(cursor.Kind() == clang.Cursor_StructDecl) ||
		(cursor.Kind() == clang.Cursor_ClassTemplate) ||
		(cursor.Kind() == clang.Cursor_ClassTemplatePartialSpecialization)) && (cursor.Spelling() == "") {
		return true
	}

	return IsInAnonymousOrDetailNamespaceOrClass(cursor.SemanticParent())
}

// Returns true if the given cursor is a deleted member method. Libclang does not provide a nicer way to do this.
// @llr REQ-TRAQ-SWL-61
func IsDeleted(cursor clang.Cursor) bool {
	// There is no actual way to check for deleted functions, but this should be close enough...
	return cursor.IsFunctionInlined() && cursor.Definition().IsNull() && !cursor.CXXMethod_IsDefaulted()
}

// Traverses the AST obtained from libclang to find any code and returns a map of files to a map of lines to code tags
// @llr REQ-TRAQ-SWL-61, REQ-TRAQ-SWL-62, REQ-TRAQ-SWL-63, REQ-TRAQ-SWL-69
func visitAstNodes(cursor clang.Cursor, repoName repos.RepoName, repoPath string, path string, fileMap map[string]struct{}) map[string]map[uint]*Code {
	code := map[string]map[uint]*Code{}

	storeTag := func(cursor clang.Cursor, optional bool) {
		file, line, _, _ := cursor.Location().FileLocation()

		// Try to get relative path to the repo
		relativePath, err := filepath.Rel(repoPath, file.TryGetRealPathName())
		if err != nil {
			// Path not in repo, continue
			return
		}

		// Only save it if we are interested in data from this file (appears in fileMap)
		if _, ok := fileMap[relativePath]; !ok {
			return
		}

		if _, ok := code[relativePath]; !ok {
			code[relativePath] = make(map[uint]*Code)
		}

		code[relativePath][uint(line)] = &Code{
			CodeFile: CodeFile{
				RepoName: repoName,
				Path:     relativePath,
			},
			Tag:      cursor.Spelling(),
			Symbol:   cursor.USR(),
			Line:     int(line),
			Optional: optional,
		}
	}

	cursor.Visit(func(cursor, parent clang.Cursor) clang.ChildVisitResult {
		if cursor.IsNull() {
			return clang.ChildVisit_Continue
		}

		switch cursor.Kind() {
		case clang.Cursor_ClassDecl, clang.Cursor_EnumDecl, clang.Cursor_StructDecl, clang.Cursor_ClassTemplate, clang.Cursor_ClassTemplatePartialSpecialization:
			if !IsPublic(cursor) {
				return clang.ChildVisit_Continue
			}

			// Only recurse if there is a chance that we can find something of interest.
			// In practice if the element is not public, there is no point in recursing
			return clang.ChildVisit_Recurse

		case clang.Cursor_Namespace:
			if cursor.Spelling() == "" || cursor.Spelling() == "detail" {
				// These namespaces are excluded from the requirements, they are considered to have
				// private access specifiers
				return clang.ChildVisit_Continue
			}
			return clang.ChildVisit_Recurse

		case clang.Cursor_TypeAliasDecl, clang.Cursor_TypeAliasTemplateDecl:
			if !IsPublic(cursor) || IsInAnonymousOrDetailNamespaceOrClass(cursor) {
				return clang.ChildVisit_Continue
			}

			// type alias CAN have parent requirements but DO NOT HAVE TO.
			storeTag(cursor, true)

		case clang.Cursor_CXXMethod, clang.Cursor_FunctionDecl, clang.Cursor_FunctionTemplate, clang.Cursor_Constructor, clang.Cursor_ConversionFunction:
			if !IsPublic(cursor) || IsInAnonymousOrDetailNamespaceOrClass(cursor) || IsDeleted(cursor) || cursor.CXXMethod_IsPureVirtual() {
				return clang.ChildVisit_Continue
			}

			if strings.HasPrefix(cursor.Spelling(), "<deduction guide for ") {
				return clang.ChildVisit_Continue
			}

			// Regular functions are never optional
			storeTag(cursor, false)
		}

		return clang.ChildVisit_Continue
	})

	return code
}

// Parses a single file as a translation unit, providing tags from all included files that are listed in the file map
// @llr REQ-TRAQ-SWL-61, REQ-TRAQ-SWL-62, REQ-TRAQ-SWL-63
func parseSingleFile(index *clang.Index, codeFile CodeFile, commands clang.CompileCommands, compilerArgs []string, fileMap map[string]struct{}) (map[string]map[uint]*Code, error) {
	repoPath, err := repos.GetRepoPathByName(codeFile.RepoName)
	if err != nil {
		return map[string]map[uint]*Code{}, err
	}
	absRepoPath, err := filepath.Abs(string(repoPath))
	if err != nil {
		return map[string]map[uint]*Code{}, err
	}

	pathInRepo, err := repos.PathInRepo(codeFile.RepoName, codeFile.Path)
	if err != nil {
		return map[string]map[uint]*Code{}, err
	}

	pathInRepo, err = filepath.Abs(pathInRepo)
	if err != nil {
		return map[string]map[uint]*Code{}, err
	}

	cmdline := []string{}
	command := findMatchingCommand(pathInRepo, commands)
	buildDir := absRepoPath
	if command != nil {
		for i := uint32(0); i < command.NumArgs(); i++ {
			cmdline = append(cmdline, command.Arg(i))
		}
		buildDir = command.Directory()
	}

	// We need to check the translation unit in the build directory
	originalDir, err := os.Getwd()
	if err != nil {
		return map[string]map[uint]*Code{}, err
	}
	err = os.Chdir(buildDir)
	if err != nil {
		return map[string]map[uint]*Code{}, err
	}
	defer os.Chdir(originalDir)

	var tu clang.TranslationUnit
	var clangErr clang.ErrorCode
	if len(cmdline) != 0 {
		clangErr = index.ParseTranslationUnit2FullArgv("", cmdline, nil, 0, &tu)
	} else {
		clangErr = index.ParseTranslationUnit2(pathInRepo, compilerArgs, nil, 0, &tu)
	}
	if clangErr != clang.Error_Success {
		return map[string]map[uint]*Code{}, fmt.Errorf("Error parsing translation unit `%s`, %v\n", codeFile.Path, clangErr)
	}
	defer tu.Dispose()

	for _, d := range tu.Diagnostics() {
		fmt.Printf("Diagnostic for file %s: %s\n", codeFile.Path, d.Spelling())
	}

	return visitAstNodes(tu.TranslationUnitCursor(), codeFile.RepoName, absRepoPath, codeFile.Path, fileMap), nil

}

// Code parser that uses Clang to parse code
type ClangCodeParser struct{}

// Tags the code in the given repository using libclang. The compilationDatabase path and clang arguments are optional
// and used to provide libclang as much information as possible when parsing the code. This function will parse each file individually,
// but collect tagged data from all included files. This helps to tag code from header files that normally is
// not found in the compilation database (because it is only part of a translation unit as a result of being included from other files)
// @llr REQ-TRAQ-SWL-61, REQ-TRAQ-SWL-62, REQ-TRAQ-SWL-63
func (ClangCodeParser) tagCode(repoName repos.RepoName, codeFiles []CodeFile, compilationDatabase string, compilerArgs []string) (map[CodeFile][]*Code, error) {
	codeMap := make(map[string]map[uint]*Code)
	code := make(map[CodeFile][]*Code)

	index := clang.NewIndex(1, 0)
	defer index.Dispose()

	var compDb clang.CompilationDatabase
	if compilationDatabase != "" {
		pathInRepo, err := repos.PathInRepo(repoName, compilationDatabase)
		if err != nil {
			fmt.Printf("Warning: compilation database not found in path `%s`: `%v`\n", compilationDatabase, err)
		} else {
			var dbErr clang.CompilationDatabase_Error
			dbErr, compDb = clang.FromDirectory(filepath.Dir(pathInRepo))
			if dbErr != clang.CompilationDatabase_NoError {
				fmt.Printf("Warning: could not parse compilation database in path `%s`: `%v`\n", compilationDatabase, dbErr)
			} else {
				defer compDb.Dispose()
			}
		}
	}

	commands := compDb.AllCompileCommands()
	defer commands.Dispose()

	fileMap := make(map[string]struct{})
	for _, codeFile := range codeFiles {
		fileMap[codeFile.Path] = struct{}{}
	}

	for _, codeFile := range codeFiles {
		codeFromFile, err := parseSingleFile(&index, codeFile, commands, compilerArgs, fileMap)
		if err != nil {
			return code, err
		}

		// Merge with the existing code tags
		for filePath := range codeFromFile {
			if _, ok := codeMap[filePath]; !ok {
				codeMap[filePath] = make(map[uint]*Code)
			}

			for fileLine, tag := range codeFromFile[filePath] {
				codeMap[filePath][fileLine] = tag
			}
		}
	}

	// Turn the map into an array now that we cannot have duplicates
	for filePath := range codeMap {
		codeFile := CodeFile{
			Path:     filePath,
			RepoName: repoName,
		}

		codeForCurrentPath := []*Code{}
		for _, tag := range codeMap[filePath] {
			codeForCurrentPath = append(codeForCurrentPath, tag)
		}
		code[codeFile] = codeForCurrentPath
	}

	return code, nil
}

// Registers libclang as a code parser
// @llr REQ-TRAQ-SWL-61, REQ-TRAQ-SWL-65
func init() {
	registerCodeParser("clang", ClangCodeParser{})
}
