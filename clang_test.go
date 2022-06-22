//go:build clang

package main

import (
	"path/filepath"
	"testing"

	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/repos"
	"github.com/stretchr/testify/assert"
)

// @llr REQ-TRAQ-SWL-61, REQ-TRAQ-SWL-62, REQ-TRAQ-SWL-63
func TestTagCodeLibClang(t *testing.T) {

	repoName := repos.RepoName("libclangtest")
	repos.RegisterRepository(repoName, repos.RepoPath(filepath.Join(string(repos.BaseRepoPath()), "testdata/libclangtest")))

	codeFiles := []CodeFile{
		{RepoName: repoName, Path: "code/a.cc"},
		{RepoName: repoName, Path: "code/include/a.hh"},
		{RepoName: repoName, Path: "test/a/a_test.cc"},
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
		{"doThings", 17, ""},
		{"doMoreThings", 23, ""},
		{"Array<T, N>", 38, ""},
		{"operator[]", 45, ""},
		{"ButThisIsPublic", 59, ""},
		{"StructMethodsArePublicByDefault", 66, ""},
		{"JustAFreeFunction", 75, ""},
		{"sort", 95, ""},
		{"sort", 89, ""},
		{"cool", 113, ""},
	}
	LookFor(t, repoName, "code/include/a.hh", tags, expectedTags)

	expectedTags = []TagMatch{
		{"hiddenFunction", 9, ""},
		{"doThings", 15, ""},
		{"doMoreThings", 21, ""},
		{"allReqsCovered", 24, ""},
	}
	LookFor(t, repoName, "code/a.cc", tags, expectedTags)

	expectedTags = []TagMatch{
		{"TestDoThings", 9, ""},
		{"TestDoMoreThings", 15, ""},
		{"TestAllReqsCovered", 21, ""},
	}
	LookFor(t, repoName, "test/a/a_test.cc", tags, expectedTags)
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
