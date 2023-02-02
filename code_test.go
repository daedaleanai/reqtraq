package main

import (
	"fmt"
	"log"
	"path/filepath"
	"regexp"
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
	tag      string
	line     int
	links    []ReqLink
	optional bool
}

// @llr REQ-TRAQ-SWL-8, REQ-TRAQ-SWL-9
func LookFor(t *testing.T, repoName repos.RepoName, sourceFile string, codeType CodeType, tagsPerFile map[CodeFile][]*Code, expectedTags []TagMatch) {
	codeFile := CodeFile{
		Path:     sourceFile,
		RepoName: repoName,
		Type:     codeType,
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
				assert.Equal(t, codeFile, tag.CodeFile)
				assert.Equal(t, e.links, tag.Links)
				break
			}
		}
		assert.True(t, found, tag)
	}
}

// @llr REQ-TRAQ-SWL-8
func TestTagCode(t *testing.T) {

	repoName := repos.RepoName("cproject1")
	repos.RegisterRepository(repoName, repos.RepoPath(filepath.Join(string(repos.BaseRepoPath()), "testdata/cproject1")))

	tags, err := CtagsCodeParser{}.tagCode(repoName, []CodeFile{{Path: "a.cc", RepoName: repoName, Type: CodeTypeTests}}, "", []string{})
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, 1, len(tags))

	expectedTags := []TagMatch{
		{"SeparateCommentsForLLrs",
			41,
			nil, false},
		{"operator []",
			37,
			nil, false},
		{"enumerateObjects",
			27,
			nil, false},
		{"getSegment",
			17,
			nil, false},
		{"getNumberOfSegments",
			13,
			nil, false},
	}
	LookFor(t, repoName, "a.cc", CodeTypeTests, tags, expectedTags)
}

// @llr REQ-TRAQ-SWL-8, REQ-TRAQ-SWL-9, REQ-TRAQ-SWL-75
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
			TestFiles:  []string{"testdata/a.c"},
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
			[]ReqLink{
				{
					Id: "REQ-TEST-SWL-15",
					Range: Range{
						Start: Position{Line: 39, Character: 8},
						End:   Position{Line: 39, Character: 23},
					},
				},
				{
					Id: "REQ-TEST-SWL-13",
					Range: Range{
						Start: Position{Line: 38, Character: 8},
						End:   Position{Line: 38, Character: 23},
					},
				},
			},
			false},
		{"operator []",
			37,
			[]ReqLink{
				{
					Id: "REQ-TEST-SWL-13",
					Range: Range{
						Start: Position{Line: 35, Character: 8},
						End:   Position{Line: 35, Character: 23},
					},
				},
				{
					Id: "REQ-TEST-SWL-14",
					Range: Range{
						Start: Position{Line: 35, Character: 25},
						End:   Position{Line: 35, Character: 40},
					},
				},
			},
			false},
		{"enumerateObjects",
			27,
			[]ReqLink{
				{
					Id: "REQ-TEST-SWL-13",
					Range: Range{
						Start: Position{Line: 24, Character: 8},
						End:   Position{Line: 24, Character: 23},
					},
				},
			},
			false},
		{"getSegment",
			17,
			[]ReqLink{{
				Id: "REQ-TEST-SWL-12",
				Range: Range{
					Start: Position{Line: 15, Character: 8},
					End:   Position{Line: 15, Character: 23},
				},
			}},
			false},
		{"getNumberOfSegments",
			13,
			[]ReqLink{{
				Id: "REQ-TEST-SWH-11",
				Range: Range{
					Start: Position{Line: 11, Character: 8},
					End:   Position{Line: 11, Character: 23},
				},
			}}, false},
	}
	LookFor(t, repoName, "a.cc", CodeTypeImplementation, rg.CodeTags, expectedTags)

	expectedTestTags := []TagMatch{
		{"testThatSomethingHappens",
			14,
			[]ReqLink{{
				Id: "REQ-TEST-SWL-13",
				Range: Range{
					Start: Position{Line: 12, Character: 8},
					End:   Position{Line: 12, Character: 23},
				},
			}},
			true},
		{"getSegment",
			17,
			[]ReqLink{}, true},
		{"enumerateObjects",
			26,
			[]ReqLink{}, true},
	}
	LookFor(t, repoName, "testdata/a.c", CodeTypeTests, rg.CodeTags, expectedTestTags)

	rg.Reqs["REQ-TEST-SWL-13"] = &Req{ID: "REQ-TEST-SWL-13", Document: &doc, RepoName: "cproject1", Position: 12}
	rg.Reqs["REQ-TEST-SWH-11"] = &Req{ID: "REQ-TEST-SWH-11", Document: &doc, RepoName: "cproject1", Position: 13}
	rg.Reqs["REQ-TEST-SWL-15"] = &Req{ID: "REQ-TEST-SWL-15", Document: &doc, RepoName: "cproject1", Position: 14}

	errs := rg.resolve()
	assert.ElementsMatch(t,
		errs,
		[]Issue{
			{
				Path:     "a.cc",
				RepoName: "cproject1",
				Line:     13,
				Error:    fmt.Errorf("Invalid reference in function getNumberOfSegments@a.cc:13 in repo `cproject1`, `REQ-TEST-SWH-11` does not match requirement format in document `path/to/doc.md`."),
				Severity: IssueSeverityMajor,
				Type:     IssueTypeInvalidRequirementInCode,
			},
			{
				Path:     "a.cc",
				RepoName: "cproject1",
				Line:     17,
				Error:    fmt.Errorf("Invalid reference in function getSegment@a.cc:17 in repo `cproject1`, REQ-TEST-SWL-12 does not exist."),
				Severity: IssueSeverityMajor,
				Type:     IssueTypeInvalidRequirementInCode,
			},
			{
				Path:     "path/to/doc.md",
				RepoName: "cproject1",
				Line:     13,
				Error:    fmt.Errorf("Requirement `REQ-TEST-SWH-11` in document `path/to/doc.md` does not match required regexp `REQ-TEST-SWL-(\\d+)`"),
				Severity: IssueSeverityMajor,
				Type:     IssueTypeInvalidRequirementId,
			},
			{
				Path:     "a.cc",
				RepoName: "cproject1",
				Line:     37,
				Error:    fmt.Errorf("Invalid reference in function operator []@a.cc:37 in repo `cproject1`, REQ-TEST-SWL-14 does not exist."),
				Severity: IssueSeverityMajor,
				Type:     IssueTypeInvalidRequirementInCode,
			},
			{
				Path:     "path/to/doc.md",
				RepoName: "cproject1",
				Line:     13,
				Error:    fmt.Errorf("Requirement REQ-TEST-SWH-11 is not tested."),
				Severity: IssueSeverityNote,
				Type:     IssueTypeReqNotTested,
			},
			{
				Path:     "path/to/doc.md",
				RepoName: "cproject1",
				Line:     14,
				Error:    fmt.Errorf("Requirement REQ-TEST-SWL-15 is not tested."),
				Severity: IssueSeverityNote,
				Type:     IssueTypeReqNotTested,
			},
		})
}
