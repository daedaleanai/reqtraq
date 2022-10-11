//go:build clang

package main

import (
	"path/filepath"
	"testing"

	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/repos"
	"github.com/stretchr/testify/assert"
)

// @llr REQ-TRAQ-SWL-61, REQ-TRAQ-SWL-62, REQ-TRAQ-SWL-63, REQ-TRAQ-SWL-67
func TestTagCodeLibClang(t *testing.T) {

	repoName := repos.RepoName("libclangtest")
	repos.RegisterRepository(repoName, repos.RepoPath(filepath.Join(string(repos.BaseRepoPath()), "testdata/libclangtest")))

	codeFiles := []CodeFile{
		{RepoName: repoName, Path: "code/a.cc", Type: CodeTypeImplementation},
		{RepoName: repoName, Path: "code/include/a.hh", Type: CodeTypeImplementation},
		{RepoName: repoName, Path: "test/a/a_test.cc", Type: CodeTypeTests},
	}

	compilerArgs := []string{
		"-std=c++20",
		"-Icode/include",
	}

	tags, err := ClangCodeParser{}.tagCode(repoName, codeFiles, "", compilerArgs)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, 3, len(tags))

	expectedTags := []TagMatch{
		{"doThings", 17, "", false},
		{"doMoreThings", 23, "", false},
		{"Array<T, N>", 38, "", false},
		{"operator[]", 45, "", false},
		{"ButThisIsPublic", 59, "", false},
		{"StructMethodsArePublicByDefault", 66, "", false},
		{"JustAFreeFunction", 75, "", false},
		{"sort", 95, "", false},
		{"sort", 89, "", false},
		{"cool", 113, "", false},
		{"JustAFreeFunction", 119, "", false},
	}
	LookFor(t, repoName, "code/include/a.hh", CodeTypeImplementation, tags, expectedTags)

	expectedTags = []TagMatch{
		{"hiddenFunction", 9, "", false},
		{"doThings", 15, "", false},
		{"doMoreThings", 21, "", false},
		{"allReqsCovered", 24, "", false},
		{"MyType", 27, "", true},
	}
	LookFor(t, repoName, "code/a.cc", CodeTypeImplementation, tags, expectedTags)

	expectedTags = []TagMatch{
		{"TestDoThings", 9, "", false},
		{"TestDoMoreThings", 15, "", false},
		{"TestAllReqsCovered", 21, "", false},
	}
	LookFor(t, repoName, "test/a/a_test.cc", CodeTypeTests, tags, expectedTags)
}

// @llr REQ-TRAQ-SWL-36
func TestValidateUsingLibClang(t *testing.T) {
	// Actually read configuration from repositories
	repos.ClearAllRepositories()
	repos.RegisterRepository(repos.RepoName("libclangtest"), repos.RepoPath("testdata/libclangtest"))

	// Make sure the child can reach the parent
	config, err := config.ParseConfig("testdata/libclangtest")
	if err != nil {
		t.Fatal(err)
	}

	actual, err := RunValidate(t, &config)
	assert.Empty(t, err, "Got unexpected error")

	expected := `Invalid reference in function operator[]@code/include/a.hh:45 in repo ` + "`" + `libclangtest` + "`" + `, REQ-TEST-SWL-12 does not exist.
WARNING. Validation failed`

	checkValidateError(t, actual, expected)
}
