//go:build clang

/*
Parses the libclang AST and collects functions which are a target for requirements tracking.
*/

package parsers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/daedaleanai/reqtraq/code"
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
func isPublic(cursor clang.Cursor) bool {
	if cursor.IsNull() {
		return true
	}

	if cursor.AccessSpecifier() != clang.AccessSpecifier_Public && cursor.AccessSpecifier() != clang.AccessSpecifier_Invalid {
		return false
	}

	return isPublic(cursor.SemanticParent())
}

// Returns true if the cursor is part of an anonymous (or detail) namespace or class
// @llr REQ-TRAQ-SWL-62
func isInAnonymousOrDetailNamespaceOrClass(cursor clang.Cursor) bool {
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

	return isInAnonymousOrDetailNamespaceOrClass(cursor.SemanticParent())
}

// Returns true if a function is defined within an abstract class
// @llr REQ-TRAQ-SWL-82
func isInAbstractClass(cursor clang.Cursor) bool {
	if cursor.IsNull() {
		return false
	}

	parent := cursor.SemanticParent()
	if parent.IsNull() {
		return false
	}

	if (parent.Kind() != clang.Cursor_ClassDecl) && (parent.Kind() != clang.Cursor_StructDecl) {
		return false
	}

	return parent.CXXRecord_IsAbstract()
}

// Returns true if the given cursor is a deleted member method. Libclang does not provide a nicer way to do this.
// @llr REQ-TRAQ-SWL-61
func isDeleted(cursor clang.Cursor) bool {
	// There is no actual way to check for deleted functions, but this should be close enough...
	return cursor.IsFunctionInlined() && cursor.Definition().IsNull() && !cursor.CXXMethod_IsDefaulted()
}

// Traverses the AST obtained from libclang to find any code and returns a map of files to a map of lines to code tags
// @llr REQ-TRAQ-SWL-61, REQ-TRAQ-SWL-62, REQ-TRAQ-SWL-63, REQ-TRAQ-SWL-69
func visitAstNodes(cursor clang.Cursor, repoName repos.RepoName, repoPath string, path string, fileMap map[string]code.CodeFile) map[string]map[uint]*code.Code {
	codeMap := map[string]map[uint]*code.Code{}

	storeTag := func(cursor clang.Cursor, optional bool) {
		if strings.TrimSpace(cursor.Spelling()) == "" {
			// Ignore empty symbols
			return
		}

		file, line, _, _ := cursor.Location().FileLocation()

		// Try to get relative path to the repo
		relativePath, err := filepath.Rel(repoPath, file.TryGetRealPathName())
		if err != nil {
			// Path not in repo, continue
			return
		}

		var codeFile code.CodeFile
		if file, ok := fileMap[relativePath]; ok {
			codeFile = file
		} else {
			// file is not in fileMap, therefore it shall be ignored
			return
		}

		if _, ok := codeMap[relativePath]; !ok {
			codeMap[relativePath] = make(map[uint]*code.Code)
		}

		codeMap[relativePath][uint(line)] = &code.Code{
			CodeFile: codeFile,
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
		case clang.Cursor_UnexposedDecl:
			// libclang exposes concepts via Cursor_UnexposedDecl
			// concepts CAN have parent requirements but DO NOT HAVE TO.
			storeTag(cursor, true)

			return clang.ChildVisit_Recurse

		case clang.Cursor_ClassDecl, clang.Cursor_EnumDecl, clang.Cursor_StructDecl, clang.Cursor_ClassTemplate, clang.Cursor_ClassTemplatePartialSpecialization:
			if !isPublic(cursor) {
				return clang.ChildVisit_Continue
			}

			// Classes, Enums, and Structs CAN have parent requirements but DO NOT HAVE TO.
			storeTag(cursor, true)

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
			if !isPublic(cursor) || isInAnonymousOrDetailNamespaceOrClass(cursor) {
				return clang.ChildVisit_Continue
			}

			// type alias CAN have parent requirements but DO NOT HAVE TO.
			storeTag(cursor, true)

		case clang.Cursor_CXXMethod, clang.Cursor_FunctionDecl, clang.Cursor_FunctionTemplate, clang.Cursor_Constructor, clang.Cursor_ConversionFunction:
			if !isPublic(cursor) || isInAnonymousOrDetailNamespaceOrClass(cursor) || isDeleted(cursor) || cursor.CXXMethod_IsPureVirtual() {
				return clang.ChildVisit_Continue
			}

			if strings.HasPrefix(cursor.Spelling(), "<deduction guide for ") {
				return clang.ChildVisit_Continue
			}

			// Regular functions are never optional
			storeTag(cursor, false)
		case clang.Cursor_Destructor:
			if !isPublic(cursor) || isInAnonymousOrDetailNamespaceOrClass(cursor) || isDeleted(cursor) {
				return clang.ChildVisit_Continue
			}

			if isInAbstractClass(cursor) {
				// Destructors in abstract classes do not need requirements
				return clang.ChildVisit_Continue
			}

			storeTag(cursor, false)
		}

		return clang.ChildVisit_Continue
	})

	return codeMap
}

// Translates a compilation command to a string array, removing arguments that
// are not needed for parsing and only affect the output (-MD and -MF)
// @llr REQ-TRAQ-SWL-61
func translateCommand(command *clang.CompileCommand) []string {
	cmdline := []string{}
	if command == nil {
		return cmdline
	}

	skipNext := false
	for i := uint32(0); i < command.NumArgs(); i++ {
		if skipNext {
			skipNext = false
			continue
		}

		if command.Arg(i) == "-MF" {
			skipNext = true
			continue
		}

		if command.Arg(i) == "-MD" {
			continue
		}

		cmdline = append(cmdline, command.Arg(i))
	}
	return cmdline
}

// Parses a single file as a translation unit, providing tags from all included files that are listed in the file map
// @llr REQ-TRAQ-SWL-61, REQ-TRAQ-SWL-62, REQ-TRAQ-SWL-63
func parseSingleFile(index *clang.Index, codeFile code.CodeFile, commands clang.CompileCommands, compilerArgs []string, fileMap map[string]code.CodeFile) (map[string]map[uint]*code.Code, error) {
	repoPath, err := repos.GetRepoPathByName(codeFile.RepoName)
	if err != nil {
		return map[string]map[uint]*code.Code{}, err
	}
	absRepoPath, err := filepath.Abs(string(repoPath))
	if err != nil {
		return map[string]map[uint]*code.Code{}, err
	}

	pathInRepo, err := repos.PathInRepo(codeFile.RepoName, codeFile.Path)
	if err != nil {
		return map[string]map[uint]*code.Code{}, err
	}

	pathInRepo, err = filepath.Abs(pathInRepo)
	if err != nil {
		return map[string]map[uint]*code.Code{}, err
	}
	fmt.Printf("Processing file: %s\n", pathInRepo)

	command := findMatchingCommand(pathInRepo, commands)
	buildDir := absRepoPath
	if command != nil {
		buildDir = command.Directory()
	}

	// We need to check the translation unit in the build directory
	originalDir, err := os.Getwd()
	if err != nil {
		return map[string]map[uint]*code.Code{}, err
	}
	err = os.Chdir(buildDir)
	if err != nil {
		return map[string]map[uint]*code.Code{}, err
	}
	defer os.Chdir(originalDir)

	var tu clang.TranslationUnit
	var clangErr clang.ErrorCode
	cmdline := translateCommand(command)
	if len(cmdline) != 0 {
		clangErr = index.ParseTranslationUnit2FullArgv("", cmdline, nil, 0, &tu)
	} else {
		clangErr = index.ParseTranslationUnit2(pathInRepo, compilerArgs, nil, 0, &tu)
	}
	if clangErr != clang.Error_Success {
		return map[string]map[uint]*code.Code{}, fmt.Errorf("Error parsing translation unit `%s`, %v\n", codeFile.Path, clangErr)
	}
	defer tu.Dispose()

	for _, d := range tu.Diagnostics() {
		fmt.Printf("Diagnostic for file %s: %s\n", codeFile.Path, d.Spelling())
	}
	if len(tu.Diagnostics()) != 0 {
		return map[string]map[uint]*code.Code{}, fmt.Errorf("Diagnostic errors parsing translation unit `%s`\n", codeFile.Path)
	}

	return visitAstNodes(tu.TranslationUnitCursor(), codeFile.RepoName, absRepoPath, codeFile.Path, fileMap), nil

}

// Code parser that uses Clang to parse code
type clangCodeParser struct{}

// Tags the code in the given repository using libclang. The compilationDatabase path and clang arguments are optional
// and used to provide libclang as much information as possible when parsing the code. This function will parse each file individually,
// but collect tagged data from all included files. This helps to tag code from header files that normally is
// not found in the compilation database (because it is only part of a translation unit as a result of being included from other files)
// @llr REQ-TRAQ-SWL-61, REQ-TRAQ-SWL-62, REQ-TRAQ-SWL-63
func (clangCodeParser) TagCode(repoName repos.RepoName, codeFiles []code.CodeFile, compilationDatabase string, compilerArgs []string) (map[code.CodeFile][]*code.Code, error) {
	codeMap := make(map[string]map[uint]*code.Code)
	tagsPerFile := make(map[code.CodeFile][]*code.Code)

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

	fileMap := make(map[string]code.CodeFile)
	for _, codeFile := range codeFiles {
		fileMap[codeFile.Path] = codeFile
	}

	for _, codeFile := range codeFiles {
		codeFromFile, err := parseSingleFile(&index, codeFile, commands, compilerArgs, fileMap)
		if err != nil {
			return tagsPerFile, err
		}

		// Merge with the existing code tags
		for filePath := range codeFromFile {
			if _, ok := codeMap[filePath]; !ok {
				codeMap[filePath] = make(map[uint]*code.Code)
			}

			for fileLine, tag := range codeFromFile[filePath] {
				codeMap[filePath][fileLine] = tag
			}
		}
	}

	// Turn the map into an array now that we cannot have duplicates
	for filePath := range codeMap {
		codeFile := fileMap[filePath]

		codeForCurrentPath := []*code.Code{}
		for _, tag := range codeMap[filePath] {
			codeForCurrentPath = append(codeForCurrentPath, tag)
		}
		tagsPerFile[codeFile] = codeForCurrentPath
	}

	return tagsPerFile, nil
}

// Registers libclang as a code parser
// @llr REQ-TRAQ-SWL-61, REQ-TRAQ-SWL-65
func init() {
	code.RegisterCodeParser("clang", clangCodeParser{})
}
