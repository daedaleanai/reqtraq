//go:build clang

package cmd

import (
	"testing"

	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/repos"
)

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

	expected := `Invalid reference in function operator[]@code/include/a.hh:45 in repo ` + "`" + `libclangtest` + "`" + `, REQ-TEST-SWL-12 does not exist.
LLR declarations differ in doThings@code/include/a.hh:134 and doThings@code/a.cc:16.`

	checkValidate(t, &config, expected, "")
}
