package code

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/repos"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

// Other packages (config) are expected to do this, but for the repos config we can do it here
// @llr REQ-TRAQ-SWL-49
func TestMain(m *testing.M) {
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatal("Could not get current directory")
	}

	repos.SetBaseRepoInfo(repos.RepoPath(filepath.Dir(workingDir)), repos.RepoName("reqtraq"))
	os.Exit(m.Run())
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

	codeTags, err := ParseCode(repoName, &doc)
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
	LookFor(t, repoName, "a.cc", CodeTypeImplementation, codeTags, expectedTags)

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
	LookFor(t, repoName, "testdata/a.c", CodeTypeTests, codeTags, expectedTestTags)
}
