package parsers

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/daedaleanai/reqtraq/code"
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
	workingDir = filepath.Dir(filepath.Dir(workingDir))

	Register()
	repos.SetBaseRepoInfo(repos.RepoPath(workingDir), repos.RepoName("reqtraq"))
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
	links    []code.ReqLink
	optional bool
}

// @llr REQ-TRAQ-SWL-8, REQ-TRAQ-SWL-9
func LookFor(t *testing.T, repoName repos.RepoName, sourceFile string, codeType code.CodeType, tagsPerFile map[code.CodeFile][]*code.Code, expectedTags []TagMatch) {
	codeFile := code.CodeFile{
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

	tags, err := ctagsCodeParser{}.TagCode(repoName, []code.CodeFile{{Path: "a.cc", RepoName: repoName, Type: code.CodeTypeTests}, {Path: "testdata/a.robot", RepoName: repoName, Type: code.CodeTypeTests}}, "", []string{})
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, 2, len(tags))

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
	LookFor(t, repoName, "a.cc", code.CodeTypeTests, tags, expectedTags)

	expectedRobotTags := []TagMatch{
		{"A Robot Test Case",
			16,
			nil, false},
		{"Another Robot Test Case",
			21,
			nil, false},
	}
	LookFor(t, repoName, "testdata/a.robot", code.CodeTypeTests, tags, expectedRobotTags)
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
		Implementation: []config.Implementation{
			{
				ArchImplementation: config.ArchImplementation{
					CodeFiles: []string{"a.cc"},
					TestFiles: []string{"testdata/a.c"},
				},
				CodeParser: "ctags",
			},
		},
	}

	codeTags, err := code.ParseCode(repoName, &doc)
	if !assert.NoError(t, err) {
		return
	}

	expectedTags := []TagMatch{
		{"SeparateCommentsForLLrs",
			41,
			[]code.ReqLink{
				{
					Id: "REQ-TEST-SWL-15",
					Range: code.Range{
						Start: code.Position{Line: 39, Character: 8},
						End:   code.Position{Line: 39, Character: 23},
					},
				},
				{
					Id: "REQ-TEST-SWL-13",
					Range: code.Range{
						Start: code.Position{Line: 38, Character: 8},
						End:   code.Position{Line: 38, Character: 23},
					},
				},
			},
			false},
		{"operator []",
			37,
			[]code.ReqLink{
				{
					Id: "REQ-TEST-SWL-13",
					Range: code.Range{
						Start: code.Position{Line: 35, Character: 8},
						End:   code.Position{Line: 35, Character: 23},
					},
				},
				{
					Id: "REQ-TEST-SWL-14",
					Range: code.Range{
						Start: code.Position{Line: 35, Character: 25},
						End:   code.Position{Line: 35, Character: 40},
					},
				},
			},
			false},
		{"enumerateObjects",
			27,
			[]code.ReqLink{
				{
					Id: "REQ-TEST-SWL-13",
					Range: code.Range{
						Start: code.Position{Line: 24, Character: 8},
						End:   code.Position{Line: 24, Character: 23},
					},
				},
			},
			false},
		{"getSegment",
			17,
			[]code.ReqLink{{
				Id: "REQ-TEST-SWL-12",
				Range: code.Range{
					Start: code.Position{Line: 15, Character: 8},
					End:   code.Position{Line: 15, Character: 23},
				},
			}},
			false},
		{"getNumberOfSegments",
			13,
			[]code.ReqLink{{
				Id: "REQ-TEST-SWH-11",
				Range: code.Range{
					Start: code.Position{Line: 11, Character: 8},
					End:   code.Position{Line: 11, Character: 23},
				},
			}}, false},
	}
	LookFor(t, repoName, "a.cc", code.CodeTypeImplementation, codeTags, expectedTags)

	expectedTestTags := []TagMatch{
		{"testThatSomethingHappens",
			14,
			[]code.ReqLink{{
				Id: "REQ-TEST-SWL-13",
				Range: code.Range{
					Start: code.Position{Line: 12, Character: 8},
					End:   code.Position{Line: 12, Character: 23},
				},
			}},
			true},
		{"getSegment",
			17,
			[]code.ReqLink{}, true},
		{"enumerateObjects",
			26,
			[]code.ReqLink{}, true},
	}
	LookFor(t, repoName, "testdata/a.c", code.CodeTypeTests, codeTags, expectedTestTags)
}
