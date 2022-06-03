package config

import (
	"regexp"
	"testing"

	"github.com/daedaleanai/reqtraq/repos"
	"github.com/stretchr/testify/assert"
)

// @llr REQ-TRAQ-SWL-52, REQ-TRAQ-SWL-53, REQ-TRAQ-SWL-56
func TestConfig_ParseConfig(t *testing.T) {
	repos.ClearAllRepositories()
	repos.RegisterRepository(repos.RepoName("projectA"), repos.RepoPath("../testdata/projectA"))
	repos.RegisterRepository(repos.RepoName("projectB"), repos.RepoPath("../testdata/projectB"))

	// Make sure the child can reach the parent
	config, err := ParseConfig("../testdata/projectB")
	if err != nil {
		t.Fatal(err)
	}

	commonAttributes := map[string]*Attribute{
		"RATIONALE": {
			Type:  AttributeAny,
			Value: regexp.MustCompile(".*"),
		},
		"VERIFICATION": {
			Type:  AttributeRequired,
			Value: regexp.MustCompile("(Demonstration|Unit [Tt]est|[Tt]est)"),
		},
	}

	assert.Contains(t, config.Repos, repos.RepoName("projectA"))
	assert.Contains(t, config.Repos, repos.RepoName("projectB"))
	assert.Equal(t, len(config.Repos), 2)

	assert.ElementsMatch(t, config.Repos["projectA"].Documents, []Document{
		{
			Path: "TEST-100-ORD.md",
			ReqSpec: ReqSpec{
				Prefix: ReqPrefix("TEST"),
				Level:  ReqLevel("SYS"),
			},
			Schema: Schema{
				Requirements: regexp.MustCompile(`(REQ|ASM)-TEST-SYS-(\d+)`),
				Attributes: map[string]*Attribute{
					"RATIONALE":    commonAttributes["RATIONALE"],
					"VERIFICATION": commonAttributes["VERIFICATION"],
				},
			},
			Implementation: Implementation{
				CodeFiles: []string{},
				TestFiles: []string{},
			},
		},
		{
			Path: "TEST-137-SRD.md",
			ReqSpec: ReqSpec{
				Prefix: ReqPrefix("TEST"),
				Level:  ReqLevel("SWH"),
			},
			ParentReqSpec: ReqSpec{
				Prefix: ReqPrefix("TEST"),
				Level:  ReqLevel("SYS"),
			},
			Schema: Schema{
				Requirements: regexp.MustCompile(`(REQ|ASM)-TEST-SWH-(\d+)`),
				Attributes: map[string]*Attribute{
					"RATIONALE":    commonAttributes["RATIONALE"],
					"VERIFICATION": commonAttributes["VERIFICATION"],
					"PARENTS": {
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
	assert.Equal(t, config.Repos["projectB"].Documents[0].ReqSpec.Prefix, ReqPrefix("TEST"))
	assert.Equal(t, config.Repos["projectB"].Documents[0].ReqSpec.Level, ReqLevel("SWL"))
	assert.Equal(t, config.Repos["projectB"].Documents[0].ParentReqSpec, ReqSpec{Prefix: ReqPrefix("TEST"), Level: ReqLevel("SWH")})
	assert.Equal(t, config.Repos["projectB"].Documents[0].Schema.Requirements, regexp.MustCompile(`(REQ|ASM)-TEST-SWL-(\d+)`))
	assert.Equal(t, len(config.Repos["projectB"].Documents[0].Schema.Attributes), 3)
	assert.Equal(t, *config.Repos["projectB"].Documents[0].Schema.Attributes["PARENTS"],
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
