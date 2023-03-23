package config

import (
	"regexp"
	"testing"

	"github.com/daedaleanai/reqtraq/repos"
	"github.com/stretchr/testify/assert"
)

// @llr REQ-TRAQ-SWL-52, REQ-TRAQ-SWL-53, REQ-TRAQ-SWL-56
func TestConfig_ParseConfig(t *testing.T) {
	DirectDependenciesOnly = false
	repos.ClearAllRepositories()
	repos.RegisterRepository(repos.RepoName("projectA"), repos.RepoPath("../testdata/projectA"))
	repos.RegisterRepository(repos.RepoName("projectB"), repos.RepoPath("../testdata/projectB"))
	repos.RegisterRepository(repos.RepoName("projectC"), repos.RepoPath("../testdata/projectC"))

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
		"SAFETY IMPACT": {
			Type:  AttributeRequired,
			Value: regexp.MustCompile(".*"),
		},
	}

	assert.Contains(t, config.Repos, repos.RepoName("projectA"))
	assert.Contains(t, config.Repos, repos.RepoName("projectB"))
	assert.Contains(t, config.Repos, repos.RepoName("projectC"))
	assert.Equal(t, len(config.Repos), 3)

	assert.ElementsMatch(t, config.Repos["projectA"].Documents, []Document{
		{
			Path: "TEST-100-ORD.md",
			ReqSpec: ReqSpec{
				Prefix: ReqPrefix("TEST"),
				Level:  ReqLevel("SYS"),
			},
			LinkSpecs: nil,
			Schema: Schema{
				Requirements: regexp.MustCompile(`(REQ|ASM)-TEST-SYS-(\d+)`),
				Attributes: map[string]*Attribute{
					"RATIONALE":     commonAttributes["RATIONALE"],
					"VERIFICATION":  commonAttributes["VERIFICATION"],
					"SAFETY IMPACT": commonAttributes["SAFETY IMPACT"],
				},
				AsmAttributes: map[string]*Attribute{
					"PARENTS": {
						Value: regexp.MustCompile("REQ-TEST-SYS-(\\d+)"),
						Type:  AttributeRequired,
					},
				},
			},
			Implementation: Implementation{
				CodeFiles:           []string{},
				TestFiles:           []string{},
				CodeParser:          "ctags",
				CompilationDatabase: "",
				CompilerArguments:   []string{},
			},
		},
		{
			Path: "TEST-137-SRD.md",
			ReqSpec: ReqSpec{
				Prefix: ReqPrefix("TEST"),
				Level:  ReqLevel("SWH"),
			},
			LinkSpecs: []LinkSpec{
				{
					Src: ReqSpec{
						Prefix:  ReqPrefix("TEST"),
						Level:   ReqLevel("SWH"),
						Re:      regexp.MustCompile("REQ-TEST-SWH-(\\d+)"),
						AttrKey: "",
						AttrVal: regexp.MustCompile(".*")},
					Dst: ReqSpec{
						Prefix:  ReqPrefix("TEST"),
						Level:   ReqLevel("SYS"),
						Re:      regexp.MustCompile("REQ-TEST-SYS-(\\d+)"),
						AttrKey: "",
						AttrVal: regexp.MustCompile(".*")},
				},
			},
			Schema: Schema{
				Requirements: regexp.MustCompile(`(REQ|ASM)-TEST-SWH-(\d+)`),
				Attributes: map[string]*Attribute{
					"RATIONALE":     commonAttributes["RATIONALE"],
					"VERIFICATION":  commonAttributes["VERIFICATION"],
					"SAFETY IMPACT": commonAttributes["SAFETY IMPACT"],
					"PARENTS": {
						Value: regexp.MustCompile(`.*`),
						Type:  AttributeAny,
					},
				},
				AsmAttributes: map[string]*Attribute{
					"PARENTS": {
						Value: regexp.MustCompile("REQ-TEST-SWH-(\\d+)"),
						Type:  AttributeRequired,
					},
					"VALIDATION": {
						Value: regexp.MustCompile(".*"),
						Type:  AttributeRequired,
					},
				},
			},
			Implementation: Implementation{
				CodeFiles:           []string{},
				TestFiles:           []string{},
				CodeParser:          "ctags",
				CompilationDatabase: "",
				CompilerArguments:   []string{},
			},
		},
	})

	assert.Equal(t, len(config.Repos["projectB"].Documents), 1)
	assert.Equal(t, config.Repos["projectB"].Documents[0].Path, "TEST-138-SDD.md")
	assert.Equal(t, config.Repos["projectB"].Documents[0].ReqSpec.Prefix, ReqPrefix("TEST"))
	assert.Equal(t, config.Repos["projectB"].Documents[0].ReqSpec.Level, ReqLevel("SWL"))
	assert.Equal(t, config.Repos["projectB"].Documents[0].LinkSpecs, []LinkSpec{
		{
			Src: ReqSpec{
				Prefix:  ReqPrefix("TEST"),
				Level:   ReqLevel("SWL"),
				Re:      regexp.MustCompile("REQ-TEST-SWL-(\\d+)"),
				AttrKey: "",
				AttrVal: regexp.MustCompile(".*")},
			Dst: ReqSpec{
				Prefix:  ReqPrefix("TEST"),
				Level:   ReqLevel("SWH"),
				Re:      regexp.MustCompile("REQ-TEST-SWH-(\\d+)"),
				AttrKey: "",
				AttrVal: regexp.MustCompile(".*")},
		},
	})
	assert.Equal(t, config.Repos["projectB"].Documents[0].Schema.Requirements, regexp.MustCompile(`(REQ|ASM)-TEST-SWL-(\d+)`))
	assert.Equal(t, config.Repos["projectB"].Documents[0].Schema.Attributes, map[string]*Attribute{
		"RATIONALE":     commonAttributes["RATIONALE"],
		"VERIFICATION":  commonAttributes["VERIFICATION"],
		"SAFETY IMPACT": commonAttributes["SAFETY IMPACT"],
		"PARENTS": {
			Value: regexp.MustCompile(`.*`),
			Type:  AttributeAny,
		},
	})
	assert.Equal(t, *config.Repos["projectB"].Documents[0].Schema.AsmAttributes["PARENTS"],
		Attribute{
			Value: regexp.MustCompile("REQ-TEST-SWL-(\\d+)"),
			Type:  AttributeRequired,
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

	assert.Equal(t, len(config.Repos["projectC"].Documents), 1)
	assert.Equal(t, config.Repos["projectC"].Documents[0].Path, "TST-138-SDD.md")
	assert.Equal(t, config.Repos["projectC"].Documents[0].ReqSpec.Prefix, ReqPrefix("TST"))
	assert.Equal(t, config.Repos["projectC"].Documents[0].ReqSpec.Level, ReqLevel("SWL"))
	assert.Equal(t, config.Repos["projectC"].Documents[0].LinkSpecs, []LinkSpec{
		{
			Src: ReqSpec{
				Prefix:  ReqPrefix("TST"),
				Level:   ReqLevel("SWL"),
				Re:      regexp.MustCompile("REQ-TST-SWL-(\\d+)"),
				AttrKey: "",
				AttrVal: regexp.MustCompile(".*")},
			Dst: ReqSpec{
				Prefix:  ReqPrefix("TEST"),
				Level:   ReqLevel("SWH"),
				Re:      regexp.MustCompile("REQ-TEST-SWH-(\\d+)"),
				AttrKey: "",
				AttrVal: regexp.MustCompile(".*")},
		},
	})
	assert.Equal(t, config.Repos["projectC"].Documents[0].Schema.Requirements, regexp.MustCompile(`(REQ|ASM)-TST-SWL-(\d+)`))
	assert.Equal(t, config.Repos["projectC"].Documents[0].Schema.Attributes, map[string]*Attribute{
		"RATIONALE":     commonAttributes["RATIONALE"],
		"VERIFICATION":  commonAttributes["VERIFICATION"],
		"SAFETY IMPACT": commonAttributes["SAFETY IMPACT"],
		"PARENTS": {
			Value: regexp.MustCompile(`.*`),
			Type:  AttributeAny,
		},
	})
	assert.Equal(t, *config.Repos["projectC"].Documents[0].Schema.AsmAttributes["PARENTS"],
		Attribute{
			Value: regexp.MustCompile("REQ-TST-SWL-(\\d+)"),
			Type:  AttributeRequired,
		},
	)
	assert.ElementsMatch(t, config.Repos["projectC"].Documents[0].Implementation.CodeFiles,
		[]string{
			"code/a.cc",
			"code/include/a.hh",
		},
	)

	assert.ElementsMatch(t, config.Repos["projectC"].Documents[0].Implementation.TestFiles,
		[]string{
			"test/a/a_test.cc",
		},
	)
}

// @llr REQ-TRAQ-SWL-52, REQ-TRAQ-SWL-53, REQ-TRAQ-SWL-56, REQ-TRAQ-SWL-68
func TestConfig_ParseConfigOnlyDirectDeps(t *testing.T) {
	DirectDependenciesOnly = true
	repos.ClearAllRepositories()
	repos.RegisterRepository(repos.RepoName("projectA"), repos.RepoPath("../testdata/projectA"))
	repos.RegisterRepository(repos.RepoName("projectB"), repos.RepoPath("../testdata/projectB"))

	// Make sure the child can reach the parent
	parsedConfig, err := ParseConfig("../testdata/projectB")
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
		"SAFETY IMPACT": {
			Type:  AttributeRequired,
			Value: regexp.MustCompile(".*"),
		},
	}

	assert.Contains(t, parsedConfig.Repos, repos.RepoName("projectA"))
	assert.Contains(t, parsedConfig.Repos, repos.RepoName("projectB"))
	assert.Equal(t, len(parsedConfig.Repos), 2)

	assert.ElementsMatch(t, parsedConfig.Repos["projectA"].Documents, []Document{
		{
			Path: "TEST-100-ORD.md",
			ReqSpec: ReqSpec{
				Prefix: ReqPrefix("TEST"),
				Level:  ReqLevel("SYS"),
			},
			LinkSpecs: nil,
			Schema: Schema{
				Requirements: regexp.MustCompile(`(REQ|ASM)-TEST-SYS-(\d+)`),
				Attributes: map[string]*Attribute{
					"RATIONALE":     commonAttributes["RATIONALE"],
					"VERIFICATION":  commonAttributes["VERIFICATION"],
					"SAFETY IMPACT": commonAttributes["SAFETY IMPACT"],
				},
				AsmAttributes: map[string]*Attribute{
					"PARENTS": {
						Value: regexp.MustCompile("REQ-TEST-SYS-(\\d+)"),
						Type:  AttributeRequired,
					},
				},
			},
			Implementation: Implementation{
				CodeFiles:           []string{},
				TestFiles:           []string{},
				CodeParser:          "ctags",
				CompilationDatabase: "",
				CompilerArguments:   []string{},
			},
		},
		{
			Path: "TEST-137-SRD.md",
			ReqSpec: ReqSpec{
				Prefix: ReqPrefix("TEST"),
				Level:  ReqLevel("SWH"),
			},
			LinkSpecs: []LinkSpec{
				{
					Src: ReqSpec{
						Prefix:  ReqPrefix("TEST"),
						Level:   ReqLevel("SWH"),
						Re:      regexp.MustCompile("REQ-TEST-SWH-(\\d+)"),
						AttrKey: "",
						AttrVal: regexp.MustCompile(".*")},
					Dst: ReqSpec{
						Prefix:  ReqPrefix("TEST"),
						Level:   ReqLevel("SYS"),
						Re:      regexp.MustCompile("REQ-TEST-SYS-(\\d+)"),
						AttrKey: "",
						AttrVal: regexp.MustCompile(".*")},
				},
			},
			Schema: Schema{
				Requirements: regexp.MustCompile(`(REQ|ASM)-TEST-SWH-(\d+)`),
				Attributes: map[string]*Attribute{
					"RATIONALE":     commonAttributes["RATIONALE"],
					"VERIFICATION":  commonAttributes["VERIFICATION"],
					"SAFETY IMPACT": commonAttributes["SAFETY IMPACT"],
					"PARENTS": {
						Value: regexp.MustCompile(`.*`),
						Type:  AttributeAny,
					},
				},
				AsmAttributes: map[string]*Attribute{
					"PARENTS": {
						Value: regexp.MustCompile("REQ-TEST-SWH-(\\d+)"),
						Type:  AttributeRequired,
					},
					"VALIDATION": {
						Value: regexp.MustCompile(".*"),
						Type:  AttributeRequired,
					},
				},
			},
			Implementation: Implementation{
				CodeFiles:           []string{},
				TestFiles:           []string{},
				CodeParser:          "ctags",
				CompilationDatabase: "",
				CompilerArguments:   []string{},
			},
		},
	})

	assert.Equal(t, len(parsedConfig.Repos["projectB"].Documents), 1)
	assert.Equal(t, parsedConfig.Repos["projectB"].Documents[0].Path, "TEST-138-SDD.md")
	assert.Equal(t, parsedConfig.Repos["projectB"].Documents[0].ReqSpec.Prefix, ReqPrefix("TEST"))
	assert.Equal(t, parsedConfig.Repos["projectB"].Documents[0].ReqSpec.Level, ReqLevel("SWL"))
	assert.Equal(t, parsedConfig.Repos["projectB"].Documents[0].LinkSpecs, []LinkSpec{
		{
			Src: ReqSpec{
				Prefix:  ReqPrefix("TEST"),
				Level:   ReqLevel("SWL"),
				Re:      regexp.MustCompile("REQ-TEST-SWL-(\\d+)"),
				AttrKey: "",
				AttrVal: regexp.MustCompile(".*")},
			Dst: ReqSpec{
				Prefix:  ReqPrefix("TEST"),
				Level:   ReqLevel("SWH"),
				Re:      regexp.MustCompile("REQ-TEST-SWH-(\\d+)"),
				AttrKey: "",
				AttrVal: regexp.MustCompile(".*")},
		},
	})
	assert.Equal(t, parsedConfig.Repos["projectB"].Documents[0].Schema.Requirements, regexp.MustCompile(`(REQ|ASM)-TEST-SWL-(\d+)`))
	assert.Equal(t, parsedConfig.Repos["projectB"].Documents[0].Schema.Attributes, map[string]*Attribute{
		"RATIONALE":     commonAttributes["RATIONALE"],
		"VERIFICATION":  commonAttributes["VERIFICATION"],
		"SAFETY IMPACT": commonAttributes["SAFETY IMPACT"],
		"PARENTS": {
			Value: regexp.MustCompile(`.*`),
			Type:  AttributeAny,
		},
	})
	assert.Equal(t, *parsedConfig.Repos["projectB"].Documents[0].Schema.AsmAttributes["PARENTS"],
		Attribute{
			Value: regexp.MustCompile("REQ-TEST-SWL-(\\d+)"),
			Type:  AttributeRequired,
		},
	)
	assert.ElementsMatch(t, parsedConfig.Repos["projectB"].Documents[0].Implementation.CodeFiles,
		[]string{
			"code/a.cc",
			"code/include/a.hh",
		},
	)

	assert.ElementsMatch(t, parsedConfig.Repos["projectB"].Documents[0].Implementation.TestFiles,
		[]string{
			"test/a/a_test.cc",
		},
	)

	DirectDependenciesOnly = false
}

// @llr REQ-TRAQ-SWL-52, REQ-TRAQ-SWL-53, REQ-TRAQ-SWL-56, REQ-TRAQ-SWL-64
func TestConfig_ParseConfigLibClang(t *testing.T) {
	repos.ClearAllRepositories()
	repos.RegisterRepository(repos.RepoName("libclangtest"), repos.RepoPath("../testdata/libclangtest"))

	config, err := ParseConfig("../testdata/libclangtest")
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
		"SAFETY IMPACT": {
			Type:  AttributeRequired,
			Value: regexp.MustCompile(".*"),
		},
	}

	assert.Contains(t, config.Repos, repos.RepoName("libclangtest"))
	assert.Equal(t, len(config.Repos), 1)

	assert.Contains(t, config.Repos["libclangtest"].Documents,
		Document{
			Path: "TEST-100-ORD.md",
			ReqSpec: ReqSpec{
				Prefix: ReqPrefix("TEST"),
				Level:  ReqLevel("SYS"),
			},
			LinkSpecs: nil,
			Schema: Schema{
				Requirements: regexp.MustCompile(`(REQ|ASM)-TEST-SYS-(\d+)`),
				Attributes: map[string]*Attribute{
					"RATIONALE":     commonAttributes["RATIONALE"],
					"VERIFICATION":  commonAttributes["VERIFICATION"],
					"SAFETY IMPACT": commonAttributes["SAFETY IMPACT"],
				},
				AsmAttributes: map[string]*Attribute{
					"PARENTS": {
						Value: regexp.MustCompile("REQ-TEST-SYS-(\\d+)"),
						Type:  AttributeRequired,
					},
				},
			},
			Implementation: Implementation{
				CodeFiles:           []string{},
				TestFiles:           []string{},
				CodeParser:          "ctags",
				CompilationDatabase: "",
				CompilerArguments:   []string{},
			},
		})

	assert.Contains(t, config.Repos["libclangtest"].Documents,
		Document{
			Path: "TEST-137-SRD.md",
			ReqSpec: ReqSpec{
				Prefix: ReqPrefix("TEST"),
				Level:  ReqLevel("SWH"),
			},
			LinkSpecs: []LinkSpec{
				{
					Src: ReqSpec{
						Prefix:  ReqPrefix("TEST"),
						Level:   ReqLevel("SWH"),
						Re:      regexp.MustCompile("REQ-TEST-SWH-(\\d+)"),
						AttrKey: "",
						AttrVal: regexp.MustCompile(".*")},
					Dst: ReqSpec{
						Prefix:  ReqPrefix("TEST"),
						Level:   ReqLevel("SYS"),
						Re:      regexp.MustCompile("REQ-TEST-SYS-(\\d+)"),
						AttrKey: "",
						AttrVal: regexp.MustCompile(".*")},
				},
			},
			Schema: Schema{
				Requirements: regexp.MustCompile(`(REQ|ASM)-TEST-SWH-(\d+)`),
				Attributes: map[string]*Attribute{
					"RATIONALE":     commonAttributes["RATIONALE"],
					"VERIFICATION":  commonAttributes["VERIFICATION"],
					"SAFETY IMPACT": commonAttributes["SAFETY IMPACT"],
					"PARENTS": {
						Value: regexp.MustCompile(`.*`),
						Type:  AttributeAny,
					},
				},
				AsmAttributes: map[string]*Attribute{
					"PARENTS": {
						Value: regexp.MustCompile("REQ-TEST-SWH-(\\d+)"),
						Type:  AttributeRequired,
					},
				},
			},
			Implementation: Implementation{
				CodeFiles:           []string{},
				TestFiles:           []string{},
				CodeParser:          "ctags",
				CompilationDatabase: "",
				CompilerArguments:   []string{},
			},
		})

	assert.Equal(t, len(config.Repos["libclangtest"].Documents), 3)

	assert.Equal(t, config.Repos["libclangtest"].Documents[2].Path, "TEST-138-SDD.md")
	assert.Equal(t, config.Repos["libclangtest"].Documents[2].ReqSpec.Prefix, ReqPrefix("TEST"))
	assert.Equal(t, config.Repos["libclangtest"].Documents[2].ReqSpec.Level, ReqLevel("SWL"))
	assert.Equal(t, config.Repos["libclangtest"].Documents[2].LinkSpecs, []LinkSpec{
		{
			Src: ReqSpec{
				Prefix:  ReqPrefix("TEST"),
				Level:   ReqLevel("SWL"),
				Re:      regexp.MustCompile("REQ-TEST-SWL-(\\d+)"),
				AttrKey: "",
				AttrVal: regexp.MustCompile(".*")},
			Dst: ReqSpec{
				Prefix:  ReqPrefix("TEST"),
				Level:   ReqLevel("SWH"),
				Re:      regexp.MustCompile("REQ-TEST-SWH-(\\d+)"),
				AttrKey: "",
				AttrVal: regexp.MustCompile(".*")},
		},
	})
	assert.Equal(t, config.Repos["libclangtest"].Documents[2].Schema.Requirements, regexp.MustCompile(`(REQ|ASM)-TEST-SWL-(\d+)`))
	assert.Equal(t, config.Repos["libclangtest"].Documents[2].Schema.Attributes, map[string]*Attribute{
		"RATIONALE":     commonAttributes["RATIONALE"],
		"VERIFICATION":  commonAttributes["VERIFICATION"],
		"SAFETY IMPACT": commonAttributes["SAFETY IMPACT"],
		"PARENTS": {
			Value: regexp.MustCompile(`.*`),
			Type:  AttributeAny,
		},
	})
	assert.ElementsMatch(t, config.Repos["libclangtest"].Documents[2].Implementation.CodeFiles,
		[]string{
			"code/a.cc",
			"code/include/a.hh",
		},
	)

	assert.ElementsMatch(t, config.Repos["libclangtest"].Documents[2].Implementation.TestFiles,
		[]string{
			"test/a/a_test.cc",
		},
	)
	assert.Equal(t, config.Repos["libclangtest"].Documents[2].Implementation.CodeParser, "clang")
	assert.Equal(t, config.Repos["libclangtest"].Documents[2].Implementation.CompilationDatabase, "compile_commands.json")
	assert.Equal(t, config.Repos["libclangtest"].Documents[2].Implementation.CompilerArguments, []string{
		"-std=c++20",
		"-Icode/include",
	})
}
