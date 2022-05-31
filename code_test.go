package main

import (
	"path/filepath"
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

func LookFor(t *testing.T, sourceFile string, tagsPerFile map[string][]*Code, expectedTags []TagMatch) {
	tags, ok := tagsPerFile[sourceFile]
	assert.True(t, ok)
	assert.Equal(t, 3, len(tags))

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
	repoName := repos.RegisterCurrentRepository(filepath.Join(repos.BaseRepoPath(), "testdata/cproject1"))

	tags, err := tagCode(repoName, []string{"a.c"})
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, 1, len(tags))

	expectedTags := []TagMatch{
		{"enumerateObjects",
			30,
			""},
		{"getSegment",
			20,
			""},
		{"getNumberOfSegments",
			14,
			""},
	}
	LookFor(t, "a.c", tags, expectedTags)
}

func TestReqGraph_ParseCode(t *testing.T) {
	repoName := repos.RegisterCurrentRepository(filepath.Join(repos.BaseRepoPath(), "testdata/cproject1"))

	rg := ReqGraph{Reqs: make(map[string]*Req, 0)}
	var err error

	impl := config.Implementation{
		CodeFiles: []string{"a.c"},
		TestFiles: []string{},
	}

	rg.CodeTags, err = ParseCode(repoName, &impl)
	if !assert.NoError(t, err) {
		return
	}

	expectedTags := []TagMatch{
		{"enumerateObjects",
			30,
			`REQ-TEST-SWL-13`},
		{"getSegment",
			20,
			`REQ-TEST-SWL-12`},
		{"getNumberOfSegments",
			14,
			`REQ-TEST-SWH-11`},
	}
	LookFor(t, "a.c", rg.CodeTags, expectedTags)

	rg.Reqs["REQ-TEST-SWL-13"] = &Req{Level: config.LOW}
	rg.Reqs["REQ-TEST-SWH-11"] = &Req{Level: config.HIGH}
	errs := SortErrs(rg.resolve())
	assert.Equal(t, 2, len(errs))
	assert.Equal(t, "Invalid reference in function getNumberOfSegments@a.c:14, REQ-TEST-SWH-11 is not a low-level requirement.", errs[0])
	assert.Equal(t, "Invalid reference in function getSegment@a.c:20, REQ-TEST-SWL-12 does not exist.", errs[1])
}
