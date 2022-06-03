package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/repos"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

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
			if e.tag == tag.Tag {
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

func TestTagCode(t *testing.T) {
	repoName := repos.RepoName("cproject1")
	repos.RegisterRepository(repoName, repos.RepoPath(filepath.Join(string(repos.BaseRepoPath()), "testdata/cproject1")))

	tags, err := tagCode(repoName, []string{"a.cc"})
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, 1, len(tags))

	expectedTags := []TagMatch{
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
	rg.CodeTags, err = ParseCode(repoName, &doc)
	if !assert.NoError(t, err) {
		return
	}

	expectedTags := []TagMatch{
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
