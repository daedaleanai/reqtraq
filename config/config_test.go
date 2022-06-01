package config

import (
	"regexp"
	"testing"

	"github.com/daedaleanai/reqtraq/repos"
	"github.com/stretchr/testify/assert"
)

func TestConfig_ParseConfig(t *testing.T) {
	repos.ClearAllRepositories()
	projectA := repos.RegisterRepository("../testdata/projectA")
	assert.Equal(t, projectA, repos.RepoName("projectA"))

	projectB := repos.RegisterRepository("../testdata/projectB")
	assert.Equal(t, projectB, repos.RepoName("projectB"))

	// Make sure the child can reach the parent
	config, err := ParseConfig("../testdata/projectB")
	if err != nil {
		t.Fatal(err)
	}

	commonAttributes := map[string]*Attribute{
		"Rationale": {
			Type:  AttributeAny,
			Value: regexp.MustCompile(".*"),
		},
		"Verification": {
			Type:  AttributeRequired,
			Value: regexp.MustCompile("(Demonstration|Unit [Tt]est|[Tt]est)"),
		},
	}

	assert.Contains(t, config.Repos, repos.RepoName("projectA"))
	assert.Contains(t, config.Repos, repos.RepoName("projectB"))
	assert.Equal(t, len(config.Repos), 2)

	assert.ElementsMatch(t, config.Repos["projectA"].Documents, []Document{
		{
			Path:         "TEST-100-ORD.md",
			Schema:       Schema {
				Requirements: regexp.MustCompile(`REQ-TEST-SYS-(\d+)`),
				Attributes:   map[string]*Attribute{
					"Rationale": commonAttributes["Rationale"],
					"Verification": commonAttributes["Verification"],
				},
			},
			Implementation: Implementation{
				CodeFiles: []string{},
				TestFiles: []string{},
			},
		},
		{
			Path:         "TEST-137-SRD.md",
			Schema:       Schema {
				Requirements: regexp.MustCompile(`REQ-TEST-SWH-(\d+)`),
				Attributes:   map[string]*Attribute{
					"Rationale": commonAttributes["Rationale"],
					"Verification": commonAttributes["Verification"],
					"Parents": {
						Value: regexp.MustCompile(`REQ-TEST-SYS-(\d+)`),
						Type:  AttributeAny,
					},
				},
			},
			Implementation: Implementation{
				CodeFiles: []string{},
				TestFiles: []string{},
			},
		},
	})

	assert.Equal(t, len(config.Repos["projectB"].Documents), 1)
	assert.Equal(t, config.Repos["projectB"].Documents[0].Path, "TEST-138-SDD.md")
	assert.Equal(t, config.Repos["projectB"].Documents[0].Schema.Requirements, regexp.MustCompile(`REQ-TEST-SWL-(\d+)`))
	assert.Equal(t, len(config.Repos["projectB"].Documents[0].Schema.Attributes), 3)
	assert.Equal(t, *config.Repos["projectB"].Documents[0].Schema.Attributes["Parents"],
		Attribute{
			Value: regexp.MustCompile(`REQ-TEST-SWH-(\d+)`),
			Type:  AttributeAny,
		},
	)
	assert.ElementsMatch(t, config.Repos["projectB"].Documents[0].Implementation.CodeFiles,
		[]string{
			"code/a.cc",
			"code/include/a.hh",
		},
	)

	assert.ElementsMatch(t, config.Repos["projectB"].Documents[0].Implementation.TestFiles,
		[]string{
			"test/a/a_test.cc",
		},
	)
}
