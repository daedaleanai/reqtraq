package main

import (
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/repos"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

// Setup the configuration when init runs
// @llr REQ-TRAQ-SWL-8
func init() {
	if err := setupConfiguration(); err != nil {
		log.Fatalf("Unable to setup configuration: %s", err.Error())
	}
}

// @llr REQ-TRAQ-SWL-8
func TestCheckCtagsAvailable(t *testing.T) {
	if err := checkCtagsAvailable(); err != nil {
		t.Fatal(errors.Wrap(err, "ctags not available - REQTRAQ_CTAGS env var can be set to specify location"))
	}
}

type TagMatch struct {
	tag       string
	line      int
	parentIds string
	optional  bool
}

// @llr REQ-TRAQ-SWL-8, REQ-TRAQ-SWL-9
func LookFor(t *testing.T, repoName repos.RepoName, sourceFile string, tagsPerFile map[CodeFile][]*Code, expectedTags []TagMatch) {
	codeFile := CodeFile{
		Path:     sourceFile,
		RepoName: repoName,
	}
	tags, ok := tagsPerFile[codeFile]
	assert.True(t, ok)
	assert.Equal(t, len(expectedTags), len(tags))

	for _, tag := range tags {
		found := false
		for _, e := range expectedTags {
			if e.tag == tag.Tag && tag.Line == e.line {
				found = true
				assert.Equal(t, e.line, tag.Line)
				assert.Equal(t, e.tag, tag.Tag)
				assert.Equal(t, e.optional, tag.Optional)
				if e.parentIds != "" {
					assert.Equal(t, e.parentIds, strings.Join(tag.ParentIds, ","))
				}
				break
			}
		}
		assert.True(t, found)
	}
}

// @llr REQ-TRAQ-SWL-8
func TestTagCode(t *testing.T) {

	repoName := repos.RepoName("cproject1")
	repos.RegisterRepository(repoName, repos.RepoPath(filepath.Join(string(repos.BaseRepoPath()), "testdata/cproject1")))

	tags, err := CtagsCodeParser{}.tagCode(repoName, []CodeFile{{Path: "a.cc", RepoName: repoName}}, "", []string{})
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, 1, len(tags))

	expectedTags := []TagMatch{
		{"SeparateCommentsForLLrs",
			41,
			"", false},
		{"operator []",
			37,
			"", false},
		{"enumerateObjects",
			27,
			"", false},
		{"getSegment",
			17,
			"", false},
		{"getNumberOfSegments",
			13,
			"", false},
	}
	LookFor(t, repoName, "a.cc", tags, expectedTags)
}

// @llr REQ-TRAQ-SWL-8, REQ-TRAQ-SWL-9
func TestReqGraph_ParseCode(t *testing.T) {
	repoName := repos.RepoName("cproject1")
	repos.RegisterRepository(repoName, repos.RepoPath(filepath.Join(string(repos.BaseRepoPath()), "testdata/cproject1")))

	rg := ReqGraph{Reqs: make(map[string]*Req, 0)}

	doc := config.Document{
		Path: "path/to/doc.md",
		Schema: config.Schema{
			Requirements: regexp.MustCompile("REQ-TEST-SWL-(\\d+)"),
		},
		Implementation: config.Implementation{
			CodeFiles:  []string{"a.cc"},
			TestFiles:  []string{},
			CodeParser: "ctags",
		},
	}

	var err error
	rg.CodeTags, err = ParseCode(repoName, &doc)
	if !assert.NoError(t, err) {
		return
	}

	expectedTags := []TagMatch{
		{"SeparateCommentsForLLrs",
			41,
			"REQ-TEST-SWL-15,REQ-TEST-SWL-13", false},
		{"operator []",
			37,
			"REQ-TEST-SWL-13,REQ-TEST-SWL-14", false},
		{"enumerateObjects",
			27,
			`REQ-TEST-SWL-13`, false},
		{"getSegment",
			17,
			`REQ-TEST-SWL-12`, false},
		{"getNumberOfSegments",
			13,
			`REQ-TEST-SWH-11`, false},
	}
	LookFor(t, repoName, "a.cc", rg.CodeTags, expectedTags)

	rg.Reqs["REQ-TEST-SWL-13"] = &Req{ID: "REQ-TEST-SWL-13", Document: &doc, RepoName: "cproject1"}
	rg.Reqs["REQ-TEST-SWH-11"] = &Req{ID: "REQ-TEST-SWH-11", Document: &doc, RepoName: "cproject1"}
	rg.Reqs["REQ-TEST-SWL-15"] = &Req{ID: "REQ-TEST-SWL-15", Document: &doc, RepoName: "cproject1"}

	errs := rg.resolve()
	assert.ElementsMatch(t,
		errs,
		[]Issue{
			{
				Path:     "a.cc",
				RepoName: "cproject1",
				Line:     13,
				Error:    fmt.Errorf("Invalid reference in function getNumberOfSegments@a.cc:13 in repo `cproject1`, `REQ-TEST-SWH-11` does not match requirement format in document `path/to/doc.md`."),
				Type:     IssueTypeInvalidRequirementInCode,
			},
			{
				Path:     "a.cc",
				RepoName: "cproject1",
				Line:     17,
				Error:    fmt.Errorf("Invalid reference in function getSegment@a.cc:17 in repo `cproject1`, REQ-TEST-SWL-12 does not exist."),
				Type:     IssueTypeInvalidRequirementInCode,
			},
			{
				Path:     "path/to/doc.md",
				RepoName: "cproject1",
				Line:     0,
				Error:    fmt.Errorf("Requirement `REQ-TEST-SWH-11` in document `path/to/doc.md` does not match required regexp `REQ-TEST-SWL-(\\d+)`"),
				Type:     IssueTypeInvalidRequirementId,
			},
			{
				Path:     "a.cc",
				RepoName: "cproject1",
				Line:     37,
				Error:    fmt.Errorf("Invalid reference in function operator []@a.cc:37 in repo `cproject1`, REQ-TEST-SWL-14 does not exist."),
				Type:     IssueTypeInvalidRequirementInCode,
			},
		})
}
