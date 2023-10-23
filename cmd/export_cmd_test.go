package cmd

import (
	"os"
	"testing"

	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/repos"
	"github.com/daedaleanai/reqtraq/reqs"
	"github.com/stretchr/testify/assert"
)

// @llr REQ-TRAQ-SWL-80
func TestExport_CanBeReloaded(t *testing.T) {
	repos.ClearAllRepositories()
	repos.RegisterRepository(repos.BaseRepoName(), repos.BaseRepoPath())
	reqtraqConfig, err := config.ParseConfig(repos.BaseRepoPath())
	if err != nil {
		t.Fatal(err)
	}

	rg, err := reqs.BuildGraph(&reqtraqConfig)
	if err != nil {
		t.Fatal(err)
	}

	// Cleanup the ReqGraph so we can compare it later.
	for repo, repoConfig := range rg.ReqtraqConfig.Repos {
		for i := range repoConfig.Documents {
			rg.ReqtraqConfig.Repos[repo].Documents[i].ReqSpec = config.ReqSpec{}
			rg.ReqtraqConfig.Repos[repo].Documents[i].LinkSpecs = []config.LinkSpec{}
			rg.ReqtraqConfig.Repos[repo].Documents[i].Schema = config.Schema{}
		}
	}
	for repo, codeTags := range rg.CodeTags {
		for i := range codeTags {
			rg.CodeTags[repo][i].Document = &config.Document{}
		}
	}
	for reqID := range rg.Reqs {
		rg.Reqs[reqID].Document = &config.Document{}

	}

	file, err := os.CreateTemp("", "reqtraq-export-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())
	err = exportReqsGraph(rg, file.Name(), true)
	if err != nil {
		t.Fatal(err)
	}

	rg2, err := reqs.LoadGraphs([]string{file.Name()})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, rg.ReqtraqConfig, rg2.ReqtraqConfig)
	assert.Equal(t, rg.Issues, rg2.Issues)
	assert.Equal(t, rg.Reqs, rg2.Reqs)
	assert.Equal(t, rg.CodeTags, rg2.CodeTags)
	// The statements above are trying to be helpful in isolating the
	// differentiating element, because the ReqGraph is very large.
	assert.Equal(t, rg, rg2)
}
