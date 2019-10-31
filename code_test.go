package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/git"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestCheckCtagsAvailable(t *testing.T) {
	if err := checkCtagsAvailable(); err != nil {
		t.Fatal(errors.Wrap(err, "ctags not available - REQTRAQ_CTAGS env var can be set to specify location"))
	}
}

type TagMatch struct {
	tag     string
	line    int
	comment string
}

func LookFor(t *testing.T, sourceFile string, tagsPerFile map[string][]Code, expectedTags []TagMatch) {
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
				if e.comment != "" {
					assert.Equal(t, e.comment, strings.Join(tag.Comment, "\n"))
				}
				break
			}
		}
		assert.True(t, found)
	}
}

func TestTagCode(t *testing.T) {
	tags, err := tagCode("testdata/cproject1")
	if !assert.NoError(t, err) {
		fmt.Println(tags)
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
	LookFor(t, "testdata/cproject1/a.c", tags, expectedTags)
}

func TestReqGraph_ParseComments(t *testing.T) {
	tags, err := tagCode("testdata/cproject1")
	if !assert.NoError(t, err) {
		fmt.Println(tags)
		return
	}

	rg := reqGraph{Reqs: make(map[string]*Req, 0), CodeTags: tags}
	err = rg.parseComments(filepath.Join(git.RepoPath(), "testdata/cproject1"))
	if !assert.NoError(t, err) {
		fmt.Println(tags)
		return
	}

	expectedTags := []TagMatch{
		{"enumerateObjects",
			30,
			`// This method does stuff also.
// @llr REQ-PROJ-SWL-13
// @xlr R-1`},
		{"getSegment",
			20,
			`// This method does stuff.
// @llr REQ-PROJ-SWL-12`},
		{"getNumberOfSegments",
			14,
			`// @llr REQ-PROJ-SWH-11`},
	}
	LookFor(t, "testdata/cproject1/a.c", tags, expectedTags)

	rg.Reqs["REQ-PROJ-SWL-13"] = &Req{Level: config.LOW}
	rg.Reqs["REQ-PROJ-SWH-11"] = &Req{Level: config.HIGH}
	errs := SortErrs(rg.resolveCodeTags(nil))
	assert.Equal(t, 2, len(errs))
	assert.Equal(t, "Invalid reference in file testdata/cproject1/a.c function getNumberOfSegments: REQ-PROJ-SWH-11 must be a Software Low-Level Requirement.", errs[0])
	assert.Equal(t, "Invalid reference in file testdata/cproject1/a.c function getSegment: REQ-PROJ-SWL-12 does not exist.", errs[1])
}
