package cmd

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/daedaleanai/reqtraq/code/parsers"
	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/repos"
	"github.com/daedaleanai/reqtraq/reqs"
	"github.com/stretchr/testify/assert"
)

// Other packages (config) are expected to do this, but for the repos config we can do it here
// @llr REQ-TRAQ-SWL-49
func TestMain(m *testing.M) {
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatal("Could not get current directory")
	}
	parentDir := filepath.Dir(workingDir)
	os.Chdir(parentDir)

	parsers.Register()
	repos.SetBaseRepoInfo(repos.RepoPath(parentDir), repos.RepoName("reqtraq"))
	os.Exit(m.Run())
}

// @llr REQ-TRAQ-SWL-36
func RunValidate(t *testing.T, config *config.Config, onlyErrors bool) (string, int, int, error) {
	// create requirements graph
	rg, err := reqs.BuildGraph(config)
	assert.Empty(t, err, "Unexpected error when building requirements graph")
	// prepare capture of stdout
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	// run the command
	criticalCount, lintCount := validate(rg.Issues, onlyErrors)
	// save stdout data and reset
	w.Close()
	buf, _ := ioutil.ReadAll(r)
	os.Stdout = rescueStdout

	return string(buf), criticalCount, lintCount, err
}

// @llr REQ-TRAQ-SWL-36
func TestValidateMarkdown(t *testing.T) {
	repos.RegisterRepository(repos.BaseRepoName(), repos.BaseRepoPath())

	commonAttributes := map[string]*config.Attribute{
		"RATIONALE": {
			Type:  config.AttributeAny,
			Value: regexp.MustCompile(".*"),
		},
		"VERIFICATION": {
			Type:  config.AttributeRequired,
			Value: regexp.MustCompile("(Demonstration|Unit [Tt]est|[Tt]est)"),
		},
		"SAFETY IMPACT": {
			Type:  config.AttributeRequired,
			Value: regexp.MustCompile(".*"),
		},
	}

	config := config.Config{
		Repos: map[repos.RepoName]config.RepoConfig{
			repos.BaseRepoName(): {
				Documents: []config.Document{
					{
						Path: "testdata/TestValidateCreateReqGraphMarkdown/TEST-100-ORD.md",
						ReqSpec: config.ReqSpec{
							Prefix: "TEST",
							Level:  "SYS",
						},
						Schema: config.Schema{
							Requirements: regexp.MustCompile(`REQ-TEST-SYS-(\d+)`),
							Attributes: map[string]*config.Attribute{
								"RATIONALE":     commonAttributes["RATIONALE"],
								"VERIFICATION":  commonAttributes["VERIFICATION"],
								"SAFETY IMPACT": commonAttributes["SAFETY IMPACT"],
							},
						},
						Implementation: config.Implementation{
							CodeParser: "ctags",
						},
					},
					{
						Path: "testdata/TestValidateCreateReqGraphMarkdown/TEST-137-SRD.md",
						ReqSpec: config.ReqSpec{
							Prefix: "TEST",
							Level:  "SWH",
						},
						LinkSpecs: []config.LinkSpec{
							{
								Child: config.ReqSpec{
									Re:      regexp.MustCompile("REQ-TEST-SWH-(\\d+)"),
									AttrKey: "",
									AttrVal: regexp.MustCompile(".*")},
								Parent: config.ReqSpec{
									Re:      regexp.MustCompile("REQ-TEST-SYS-(\\d+)"),
									AttrKey: "",
									AttrVal: regexp.MustCompile(".*")},
							},
						},
						Schema: config.Schema{
							Requirements: regexp.MustCompile(`REQ-TEST-SWH-(\d+)`),
							Attributes: map[string]*config.Attribute{
								"RATIONALE":     commonAttributes["RATIONALE"],
								"VERIFICATION":  commonAttributes["VERIFICATION"],
								"SAFETY IMPACT": commonAttributes["SAFETY IMPACT"],
								"PARENTS": {
									Type:  config.AttributeAny,
									Value: regexp.MustCompile(`.*`),
								},
							},
						},
						Implementation: config.Implementation{
							CodeParser: "ctags",
						},
					},
					{
						Path: "testdata/TestValidateCreateReqGraphMarkdown/TEST-138-SDD.md",
						ReqSpec: config.ReqSpec{
							Prefix: "TEST",
							Level:  "SWL",
						},
						LinkSpecs: []config.LinkSpec{
							{
								Child: config.ReqSpec{
									Re:      regexp.MustCompile("REQ-TEST-SWL-(\\d+)"),
									AttrKey: "",
									AttrVal: regexp.MustCompile(".*")},
								Parent: config.ReqSpec{
									Re:      regexp.MustCompile("REQ-TEST-SYS-(\\d+)"),
									AttrKey: "",
									AttrVal: regexp.MustCompile(".*")},
							},
						},
						Schema: config.Schema{
							Requirements: regexp.MustCompile(`REQ-TEST-SWL-(\d+)`),
							Attributes: map[string]*config.Attribute{
								"RATIONALE":     commonAttributes["RATIONALE"],
								"VERIFICATION":  commonAttributes["VERIFICATION"],
								"SAFETY IMPACT": commonAttributes["SAFETY IMPACT"],
								"PARENTS": {
									Type:  config.AttributeAny,
									Value: regexp.MustCompile(`.*`),
								},
							},
						},
						Implementation: config.Implementation{
							CodeParser: "ctags",
						},
					},
				},
			},
		},
	}

	expected := `Incorrect requirement type for requirement REQ-TEST-SWH-3. Expected SYS, got SWH.
Incorrect project abbreviation for requirement REQ-TSET-SYS-5. Expected TEST, got TSET.
Invalid requirement sequence number for REQ-TEST-SYS-1, is duplicate.
Invalid requirement sequence number for REQ-TEST-SYS-13: missing requirements in between. Expected ID Number 9.
Requirement number cannot begin with a 0: REQ-TEST-SWL-04. Got 04.
Invalid parent of requirement REQ-TEST-SWH-9: REQ-TEST-SYS-3 does not exist.
Invalid parent of requirement REQ-TEST-SWH-10: REQ-TEST-SYS-3 does not exist.
Invalid parent of requirement REQ-TEST-SWH-11: REQ-TEST-SYS-3 does not exist.
Invalid parent of requirement REQ-TEST-SWH-8: REQ-TEST-SYS-3 does not exist.
Invalid parent of requirement REQ-TEST-SWL-2: REQ-TEST-SYS-2 is deleted.
Invalid parent of requirement REQ-TEST-SWH-2: REQ-TEST-SYS-2 is deleted.
Invalid parent of requirement REQ-TEST-SWH-4: REQ-TEST-SYS-22 does not exist.
Invalid parent of requirement REQ-TEST-SWH-5: REQ-TEST-SYS-3 does not exist.
Invalid reference to deleted requirement REQ-TEST-SYS-2 in body of REQ-TEST-SWH-11.
Invalid reference to non existent requirement REQ-TEST-SYS-22 in body of REQ-TEST-SWH-5.
Requirement 'REQ-TEST-SWH-7' is missing at least one of the attributes 'PARENTS,RATIONALE'.
Requirement 'REQ-TEST-SWH-9' is missing attribute 'SAFETY IMPACT'.
Requirement 'REQ-TEST-SWH-6' is missing at least one of the attributes 'PARENTS,RATIONALE'.
Requirement 'REQ-TEST-SWH-8' is missing attribute 'VERIFICATION'.
Requirement 'REQ-TEST-SWH-10' has invalid value 'None.' in attribute 'VERIFICATION'.
Requirement 'REQ-TEST-SWH-10' is missing attribute 'SAFETY IMPACT'.
Requirement 'REQ-TEST-SWH-14' has unknown attribute 'RANDOM'.`

	checkValidate(t, &config, expected, "")
}

// @llr REQ-TRAQ-SWL-36
func TestValidateCheckReqReferencesMarkdown(t *testing.T) {
	commonAttributes := map[string]*config.Attribute{
		"RATIONALE": {
			Type:  config.AttributeAny,
			Value: regexp.MustCompile(".*"),
		},
		"VERIFICATION": {
			Type:  config.AttributeRequired,
			Value: regexp.MustCompile("(Demonstration|Unit [Tt]est|[Tt]est)"),
		},
		"SAFETY IMPACT": {
			Type:  config.AttributeRequired,
			Value: regexp.MustCompile(".*"),
		},
	}

	config := config.Config{
		Repos: map[repos.RepoName]config.RepoConfig{
			repos.BaseRepoName(): {
				Documents: []config.Document{
					{
						Path: "testdata/TestValidateCheckReqReferencesMarkdown/TEST-100-ORD.md",
						ReqSpec: config.ReqSpec{
							Prefix: "TEST",
							Level:  "SYS",
						},
						Schema: config.Schema{
							Requirements: regexp.MustCompile(`REQ-TEST-SYS-(\d+)`),
							Attributes: map[string]*config.Attribute{
								"RATIONALE":     commonAttributes["RATIONALE"],
								"VERIFICATION":  commonAttributes["VERIFICATION"],
								"SAFETY IMPACT": commonAttributes["SAFETY IMPACT"],
							},
						},
						Implementation: config.Implementation{
							CodeParser: "ctags",
						},
					},
					{
						Path: "testdata/TestValidateCheckReqReferencesMarkdown/TEST-137-SRD.md",
						ReqSpec: config.ReqSpec{
							Prefix: "TEST",
							Level:  "SWH",
						},
						LinkSpecs: []config.LinkSpec{
							{
								Child: config.ReqSpec{
									Re:      regexp.MustCompile("REQ-TEST-SWH-(\\d+)"),
									AttrKey: "",
									AttrVal: regexp.MustCompile(".*")},
								Parent: config.ReqSpec{
									Re:      regexp.MustCompile("REQ-TEST-SYS-(\\d+)"),
									AttrKey: "",
									AttrVal: regexp.MustCompile(".*")},
							},
						},
						Schema: config.Schema{
							Requirements: regexp.MustCompile(`REQ-TEST-SWH-(\d+)`),
							Attributes: map[string]*config.Attribute{
								"RATIONALE":     commonAttributes["RATIONALE"],
								"VERIFICATION":  commonAttributes["VERIFICATION"],
								"SAFETY IMPACT": commonAttributes["SAFETY IMPACT"],
								"PARENTS": {
									Type:  config.AttributeAny,
									Value: regexp.MustCompile(`.*`),
								},
							},
						},
						Implementation: config.Implementation{
							CodeParser: "ctags",
						},
					},
				},
			},
		},
	}

	expected := `Invalid reference to non existent requirement REQ-TEST-SYS-22 in body of REQ-TEST-SWH-3.
Invalid reference to deleted requirement REQ-TEST-SYS-2 in body of REQ-TEST-SWH-4.
Requirement 'REQ-TEST-SWH-6' is missing attribute 'VERIFICATION'.
Requirement 'REQ-TEST-SWH-8' has invalid value 'gibberish.' in attribute 'VERIFICATION'.
Requirement 'REQ-TEST-SWH-7' is missing attribute 'SAFETY IMPACT'.`

	checkValidate(t, &config, expected, "")
}

func splitLines(s string) (ret []string) {
	for _, s := range strings.Split(s, "\n") {
		if s != "" {
			ret = append(ret, s)
		}
	}
	return
}

// checkValidate returns an error if validation behaves unexpectedly.
// @llr REQ-TRAQ-SWL-36
func checkValidate(t *testing.T, config *config.Config, expectedCriticalRaw, expectedLintRaw string) {
	expectedCritical := splitLines(expectedCriticalRaw)
	expectedLint := splitLines(expectedLintRaw)

	checkValidateOutput(t, config, true, expectedCritical, []string{})
	checkValidateOutput(t, config, false, expectedCritical, expectedLint)
}

func checkValidateOutput(t *testing.T, config *config.Config, onlyErrors bool, expectedCritical, expectedLint []string) {
	output, criticalCount, lintCount, err := RunValidate(t, config, onlyErrors)
	assert.Empty(t, err, "Failed to validate")
	assert.Equal(t, criticalCount, len(expectedCritical), output)
	assert.Equal(t, lintCount, len(expectedLint), output)

	reportedErrors := splitLines(output)
	expected := append(expectedCritical, expectedLint...)
	for _, m := range expected {
		found := false
		for i, e := range reportedErrors {
			if e == m {
				reportedErrors = append(reportedErrors[:i], reportedErrors[i+1:]...)
				found = true
				break
			}
		}
		assert.Truef(t, found, "One of the expected errors `%s` is missing from the reported errors:\n%s", m, output)
	}

	assert.Empty(t, reportedErrors, "Got unexpected errors")
}

// @llr REQ-TRAQ-SWL-36
func TestValidateMultipleRepos(t *testing.T) {
	// Actually read configuration from repositories
	repos.ClearAllRepositories()
	repos.RegisterRepository(repos.RepoName("projectA"), repos.RepoPath("testdata/projectA"))
	repos.RegisterRepository(repos.RepoName("projectB"), repos.RepoPath("testdata/projectB"))
	repos.RegisterRepository(repos.RepoName("projectC"), repos.RepoPath("testdata/projectC"))

	// Make sure the child can reach the parent
	config, err := config.ParseConfig("testdata/projectB")
	if err != nil {
		t.Fatal(err)
	}

	expected := `Requirement 'ASM-TEST-SWH-3' is missing attribute 'VALIDATION'.
Requirement 'ASM-TEST-SWH-3' has unknown attribute 'VERIFICATION'.
Requirement 'ASM-TEST-SWH-2' has invalid value 'REQ-TEST-SYS-2' in attribute 'PARENTS'.`

	checkValidate(t, &config, expected, "")
}

// @llr REQ-TRAQ-SWL-36
func TestValidateMultipleLevelDoc(t *testing.T) {
	// Actually read configuration from repositories
	repos.ClearAllRepositories()
	repos.RegisterRepository(repos.RepoName("multiple_level_doc"), repos.RepoPath("testdata/multiple_level_doc"))

	// Make sure the child can reach the parent
	config, err := config.ParseConfig("testdata/multiple_level_doc")
	if err != nil {
		t.Fatal(err)
	}

	expected := `Requirement 'REQ-TEST-SYS-6' has invalid parent link ID 'REQ-TEST-SYS-1'.
Requirement 'REQ-TEST-SYS-7' has invalid parent link ID 'REQ-TEST-SYS-3' with attribute value 'COMPONENT ALLOCATION'=='Component1'.
Requirement 'REQ-TEST-SWH-3' has invalid parent link ID 'REQ-TEST-SYS-1' with attribute value 'COMPONENT ALLOCATION'=='System'.`

	checkValidate(t, &config, expected, "")
}

// @llr REQ-TRAQ-SWL-84, REQ-TRAQ-SWL-85, REQ-TRAQ-SWL-86
func TestValidateDataControlFlow(t *testing.T) {
	repos.RegisterRepository(repos.BaseRepoName(), repos.BaseRepoPath())

	commonAttributes := map[string]*config.Attribute{
		"RATIONALE": {
			Type:  config.AttributeAny,
			Value: regexp.MustCompile(".*"),
		},
		"VERIFICATION": {
			Type:  config.AttributeRequired,
			Value: regexp.MustCompile("(Demonstration|Unit [Tt]est|[Tt]est)"),
		},
		"SAFETY IMPACT": {
			Type:  config.AttributeRequired,
			Value: regexp.MustCompile(".*"),
		},
	}

	config := config.Config{
		Repos: map[repos.RepoName]config.RepoConfig{
			repos.BaseRepoName(): {
				Documents: []config.Document{
					{
						Path: "testdata/TestValidateDataControlFlow/TEST-138-SDD.md",
						ReqSpec: config.ReqSpec{
							Prefix: "TEST",
							Level:  "SWL",
						},
						LinkSpecs: []config.LinkSpec{
							{
								Child: config.ReqSpec{
									Re:      regexp.MustCompile("REQ-TEST-SWL-(\\d+)"),
									AttrKey: "",
									AttrVal: regexp.MustCompile(".*")},
							},
						},
						Schema: config.Schema{
							Requirements: regexp.MustCompile(`REQ-TEST-SWL-(\d+)`),
							Attributes: map[string]*config.Attribute{
								"RATIONALE":     commonAttributes["RATIONALE"],
								"VERIFICATION":  commonAttributes["VERIFICATION"],
								"SAFETY IMPACT": commonAttributes["SAFETY IMPACT"],
								"FLOW": {
									Type:  config.AttributeAny,
									Value: regexp.MustCompile(`.*`),
								},
							},
						},
						Implementation: config.Implementation{
							CodeParser: "ctags",
						},
					},
				},
			},
		},
	}

	expected := `Duplicate data/control flow tag 'CF-TEST-2'
Unknown data/control flow tag 'CF-TEST-3' in requirement 'REQ-TEST-SWL-2'
Data/control flow tag 'CF-TEST-2' has no linked requirements
Data/control flow tag 'DF-TEST-2' has no linked requirements
Missing flow tag 'CF-TEST-3'
Invalid data/control flow tag prefix in 'DF-TST-1'
Invalid direction 'Bad' for data flow tag 'DF-TEST-4'. Allowed values are 'In', 'Out' and 'In/Out'
`

	checkValidate(t, &config, expected, "")
}
