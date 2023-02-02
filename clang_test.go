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
	}
	LookFor(t, repoName, "code/include/a.hh", CodeTypeImplementation, tags, expectedTags)

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
	LookFor(t, repoName, "code/a.cc", CodeTypeImplementation, tags, expectedTags)

	expectedTags = []TagMatch{
		{"TestDoThings", 9, nil, false},
		{"TestDoMoreThings", 15, nil, false},
		{"TestAllReqsCovered", 21, nil, false},
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
