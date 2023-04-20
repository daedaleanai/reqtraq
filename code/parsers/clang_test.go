//go:build clang

package parsers

import (
	"path/filepath"
	"testing"

	"github.com/daedaleanai/reqtraq/code"
	"github.com/daedaleanai/reqtraq/repos"
	"github.com/stretchr/testify/assert"
)

// @llr REQ-TRAQ-SWL-61, REQ-TRAQ-SWL-62, REQ-TRAQ-SWL-63, REQ-TRAQ-SWL-67
func TestTagCodeLibClang(t *testing.T) {

	repoName := repos.RepoName("libclangtest")
	repos.RegisterRepository(repoName, repos.RepoPath(filepath.Join(string(repos.BaseRepoPath()), "testdata/libclangtest")))

	codeFiles := []code.CodeFile{
		{RepoName: repoName, Path: "code/a.cc", Type: code.CodeTypeImplementation},
		{RepoName: repoName, Path: "code/include/a.hh", Type: code.CodeTypeImplementation},
		{RepoName: repoName, Path: "test/a/a_test.cc", Type: code.CodeTypeTests},
	}

	compilerArgs := []string{
		"-std=c++20",
		"-Icode/include",
	}

	tags, err := clangCodeParser{}.TagCode(repoName, codeFiles, "", compilerArgs)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, 3, len(tags))

	expectedTags := []TagMatch{
		{"SomeType", 8, nil, true},
		{"doThings", 17, nil, false},
		{"doMoreThings", 23, nil, false},
		{"Array", 26, nil, true},
		{"Array<T, N>", 38, nil, false},
		{"operator[]", 45, nil, false},
		{"ButThisIsPublic", 59, nil, false},
		{"A", 62, nil, true},
		{"StructMethodsArePublicByDefault", 66, nil, false},
		{"JustAFreeFunction", 75, nil, false},
		{"sort", 95, nil, false},
		{"sort", 89, nil, false},
		{"B", 101, nil, true},
		{"cool", 113, nil, false},
		{"JustAFreeFunction", 119, nil, false},
		{"ExternCFunc", 126, nil, false},
		{"doThings", 134, nil, false},
	}
	LookFor(t, repoName, "code/include/a.hh", code.CodeTypeImplementation, tags, expectedTags)

	expectedTags = []TagMatch{
		{"hiddenFunction", 10, nil, false},
		{"doThings", 16, nil, false},
		{"doMoreThings", 22, nil, false},
		{"allReqsCovered", 25, nil, false},
		{"MyType", 28, nil, true},
		{"MyConcept", 32, nil, true},
		{"AnotherMyConcept", 37, nil, true},
		{"externFunc", 42, nil, false},
	}
	LookFor(t, repoName, "code/a.cc", code.CodeTypeImplementation, tags, expectedTags)

	expectedTags = []TagMatch{
		{"TestDoThings", 9, nil, false},
		{"TestDoMoreThings", 15, nil, false},
		{"TestAllReqsCovered", 21, nil, false},
	}
	LookFor(t, repoName, "test/a/a_test.cc", code.CodeTypeTests, tags, expectedTags)
}
