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

	tags, err := tagCode(repoName, []string{"a.cc"})
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, 1, len(tags))

	expectedTags := []TagMatch{
		{"SeparateCommentsForLLrs",
			41,
			""},
		{"operator []",
			37,
			""},
		{"enumerateObjects",
			27,
			""},
		{"getSegment",
			17,
			""},
		{"getNumberOfSegments",
			13,
			""},
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
			CodeFiles: []string{"a.cc"},
			TestFiles: []string{},
		},
	}

	var err error
	rg.CodeTags, err = ParseCode(repoName, &doc, false)
	if !assert.NoError(t, err) {
		return
	}

	expectedTags := []TagMatch{
		{"SeparateCommentsForLLrs",
			41,
			"REQ-TEST-SWL-15,REQ-TEST-SWL-13"},
		{"operator []",
			37,
			"REQ-TEST-SWL-13,REQ-TEST-SWL-14"},
		{"enumerateObjects",
			27,
			`REQ-TEST-SWL-13`},
		{"getSegment",
			17,
			`REQ-TEST-SWL-12`},
		{"getNumberOfSegments",
			13,
			`REQ-TEST-SWH-11`},
	}
	LookFor(t, repoName, "a.cc", rg.CodeTags, expectedTags)

	rg.Reqs["REQ-TEST-SWL-13"] = &Req{ID: "REQ-TEST-SWL-13", Document: &doc}
	rg.Reqs["REQ-TEST-SWH-11"] = &Req{ID: "REQ-TEST-SWH-11", Document: &doc}
	rg.Reqs["REQ-TEST-SWL-15"] = &Req{ID: "REQ-TEST-SWL-15", Document: &doc}

	errs := rg.resolve()
	assert.ElementsMatch(t,
		errs,
		[]error{
			fmt.Errorf("Invalid reference in function getNumberOfSegments@a.cc:13 in repo `cproject1`, `REQ-TEST-SWH-11` does not match requirement format in document `path/to/doc.md`."),
			fmt.Errorf("Invalid reference in function getSegment@a.cc:17 in repo `cproject1`, REQ-TEST-SWL-12 does not exist."),
			fmt.Errorf("Requirement `REQ-TEST-SWH-11` in document `path/to/doc.md` does not match required regexp `REQ-TEST-SWL-(\\d+)`"),
			fmt.Errorf("Invalid reference in function operator []@a.cc:37 in repo `cproject1`, REQ-TEST-SWL-14 does not exist."),
		})
}

// @llr REQ-TRAQ-SWL-61, REQ-TRAQ-SWL-62, REQ-TRAQ-SWL-63
func TestTagCodeLibClang(t *testing.T) {

	repoName := repos.RepoName("libclangtest")
	repos.RegisterRepository(repoName, repos.RepoPath(filepath.Join(string(repos.BaseRepoPath()), "testdata/libclangtest")))

	codeFiles := []CodeFile{
		{RepoName: repoName, Path: "code/a.cc"},
		{RepoName: repoName, Path: "code/include/a.hh"},
		{RepoName: repoName, Path: "test/a/a_test.cc"},
	}

	clangArgs := []string{
		"-std=c++20",
		"-Icode/include",
	}

	tags, err := tagCodeLibClang(repoName, codeFiles, "", clangArgs)
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
