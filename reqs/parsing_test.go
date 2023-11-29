package reqs

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/repos"
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

// TestParseMarkdown checks that parseMarkdown finds the requirements blocks
// correctly.
// @llr REQ-TRAQ-SWL-2, REQ-TRAQ-SWL-3, REQ-TRAQ-SWL-4, REQ-TRAQ-SWL-5
func TestParseMarkdown(t *testing.T) {

	// Heading style requirements
	checkParseOk(t, `
# Title
#### REQ-TEST-SYS-5 My First Requirement
##### Heading part of a req
#### REQ-TEST-SYS-6
Content mentioning REQ-TEST-SYS-1
REQ-TEST-SYS-2
### Title2
#### REQ-TEST-SYS-7 My Last Requirement
Some more content
#### ASM-TEST-SYS-1 An assumption, not a requirement
Assumption body
`,

		[]*Flow{},
		[]*Req{
			&Req{ID: "REQ-TEST-SYS-5",
				Variant:    ReqVariantRequirement,
				IDNumber:   5,
				Title:      "My First Requirement",
				Body:       "##### Heading part of a req",
				Position:   3,
				Attributes: map[string]string{}},
			&Req{ID: "REQ-TEST-SYS-6",
				Variant:    ReqVariantRequirement,
				IDNumber:   6,
				Title:      "Content mentioning REQ-TEST-SYS-1",
				Body:       "REQ-TEST-SYS-2",
				Position:   5,
				Attributes: map[string]string{}},
			&Req{ID: "REQ-TEST-SYS-7",
				Variant:    ReqVariantRequirement,
				IDNumber:   7,
				Title:      "My Last Requirement",
				Body:       "Some more content",
				Position:   9,
				Attributes: map[string]string{}},
			&Req{ID: "ASM-TEST-SYS-1",
				Variant:    ReqVariantAssumption,
				IDNumber:   1,
				Title:      "An assumption, not a requirement",
				Body:       "Assumption body",
				Position:   11,
				Attributes: map[string]string{}},
		})

	checkParseError(t, `# REQ-TEST-SYS-5 REQ-TEST-SYS-6`, `malformed requirement title: too many IDs on line 1:`)
	checkParseError(t, `
# REQ-TEST-SYS-5
## REQ-TEST-SYS-6`,
		"requirement heading on line 3 must be at same level as requirement heading on line 2 (2 != 1):")
	checkParseError(t, `
## REQ-TEST-SYS-5
# REQ-TEST-SYS-6`,
		"requirement heading on line 3 must be at same level as requirement heading on line 2 (1 != 2):")
	checkParseError(t, `
# REQ-TEST-SYS-5
# Title`,
		"non-requirement heading on line 3 at same level as requirement heading on line 2 (1):")
	checkParseError(t, `
# Title
# REQ-TEST-SYS-5`,
		"requirement heading on line 3 at same level as previous heading on line 2 (1):")

	// Table style requirements
	checkParseOk(t, `
| ID | Title | Body |
| REQ-TEST-SYS-5 | My First Requirement | Heading part of a req |
| REQ-TEST-SYS-6 | Content mentioning REQ-TEST-SYS-1 | REQ-TEST-SYS-2 |
| REQ-TEST-SYS-7 | My Last Requirement | Some more content |
| ASM-TEST-SYS-1 | An assumption, not a requirement | Assumption body |
`,
		[]*Flow{},
		[]*Req{
			&Req{ID: "REQ-TEST-SYS-5",
				Variant:    ReqVariantRequirement,
				IDNumber:   5,
				Title:      "My First Requirement",
				Body:       "Heading part of a req",
				Position:   3,
				Attributes: map[string]string{},
			},
			&Req{ID: "REQ-TEST-SYS-6",
				Variant:    ReqVariantRequirement,
				IDNumber:   6,
				Title:      "Content mentioning REQ-TEST-SYS-1",
				Body:       "REQ-TEST-SYS-2",
				Position:   4,
				Attributes: map[string]string{}},
			&Req{ID: "REQ-TEST-SYS-7",
				Variant:    ReqVariantRequirement,
				IDNumber:   7,
				Title:      "My Last Requirement",
				Body:       "Some more content",
				Position:   5,
				Attributes: map[string]string{}},
			&Req{ID: "ASM-TEST-SYS-1",
				Variant:    ReqVariantAssumption,
				IDNumber:   1,
				Title:      "An assumption, not a requirement",
				Body:       "Assumption body",
				Position:   6,
				Attributes: map[string]string{}},
		})

	// Mixed style requirements
	checkParseOk(t, `
# Title
#### REQ-TEST-SYS-5 My First Requirement
##### Heading part of a req
| ID | Title | Body |
| REQ-TEST-SYS-6 | Content mentioning REQ-TEST-SYS-1 | REQ-TEST-SYS-2 |
| REQ-TEST-SYS-7 | My Last Requirement | Some more content |
`,
		[]*Flow{},
		[]*Req{
			&Req{ID: "REQ-TEST-SYS-5",
				Variant:    ReqVariantRequirement,
				IDNumber:   5,
				Title:      "My First Requirement",
				Body:       "##### Heading part of a req",
				Position:   3,
				Attributes: map[string]string{},
			},
			&Req{ID: "REQ-TEST-SYS-6",
				Variant:    ReqVariantRequirement,
				IDNumber:   6,
				Title:      "Content mentioning REQ-TEST-SYS-1",
				Body:       "REQ-TEST-SYS-2",
				Position:   6,
				Attributes: map[string]string{}},
			&Req{ID: "REQ-TEST-SYS-7",
				Variant:    ReqVariantRequirement,
				IDNumber:   7,
				Title:      "My Last Requirement",
				Body:       "Some more content",
				Position:   7,
				Attributes: map[string]string{}},
		},
	)
}

// TestParseMarkdown checks that parseMarkdown parse data/control flow tabless
// correctly.
// @llr REQ-TRAQ-SWL-83, REQ-TRAQ-SWL-84
func TestParseDataControlFlow(t *testing.T) {
	// Data/control flow
	checkParseOk(t, `
# Title
| Caller | Flow Tag | Callee | Description |
| --- | --- | --- | --- |
| Caller Name | CF-FLT-1 | Callee Name | Flow description |
| Caller Name | CF-FLT-2 | Callee Name | Flow description |

| Caller | Flow Tag | Callee | Direction | Description |
| --- | --- | --- | --- |
| Caller Name | DF-FLT-1 | Callee Name | In | Flow description |
| Caller Name | DF-FLT-2 | Callee Name | Out | Flow description |
| Caller Name | DF-FLT-3-DELETED | Callee Name | Out | Flow description |

#### REQ-TEST-SYS-5 My First Requirement
Body
###### Attributes:
- Flow: DF-FLT-2
`,
		[]*Flow{
			&Flow{
				ID:          "CF-FLT-1",
				Callee:      "Callee Name",
				Caller:      "Caller Name",
				Description: "Flow description",
				Position:    5,
				RepoName:    ".",
				Deleted:     false,
			},
			&Flow{
				ID:          "CF-FLT-2",
				Callee:      "Callee Name",
				Caller:      "Caller Name",
				Description: "Flow description",
				Position:    6,
				RepoName:    ".",
				Deleted:     false,
			},
			&Flow{
				ID:          "DF-FLT-1",
				Callee:      "Callee Name",
				Caller:      "Caller Name",
				Description: "Flow description",
				Position:    10,
				RepoName:    ".",
				Direction:   "In",
				Deleted:     false,
			},
			&Flow{
				ID:          "DF-FLT-2",
				Callee:      "Callee Name",
				Caller:      "Caller Name",
				Description: "Flow description",
				Position:    11,
				RepoName:    ".",
				Direction:   "Out",
				Deleted:     false,
			},
			&Flow{
				ID:          "DF-FLT-3",
				Callee:      "Callee Name",
				Caller:      "Caller Name",
				Description: "Flow description",
				Position:    12,
				RepoName:    ".",
				Direction:   "Out",
				Deleted:     true,
			},
		},
		[]*Req{
			&Req{ID: "REQ-TEST-SYS-5",
				Variant:    ReqVariantRequirement,
				IDNumber:   5,
				Title:      "My First Requirement",
				Body:       "Body",
				Position:   14,
				Attributes: map[string]string{"FLOW": "DF-FLT-2"}},
		},
	)

	checkParseError(t, `
# Title
| Caller | Flow Tag | Callee | Description |
| --- | --- | --- | --- |
| Caller Name | DF-FLT-1 | Callee Name | Flow description |
		`,
		"Invalid tag 'DF-FLT-1' on row 3 of control flow table")
	checkParseError(t, `
# Title
| Caller | Flow Tag | Callee | Direction | Description |
| --- | --- | --- | --- | --- |
| Caller Name | CF-FLT-1 | Callee Name | In | Flow description |
		`,
		"Invalid tag 'CF-FLT-1' on row 3 of data flow table")
}

// @llr REQ-TRAQ-SWL-2, REQ-TRAQ-SWL-3, REQ-TRAQ-SWL-4, REQ-TRAQ-SWL-5, REQ-TRAQ-SWL-83, REQ-TRAQ-SWL-84
func doParse(t *testing.T, content string) ([]*Req, []*Flow, error) {
	f, err := createTempFile(content, "checkParse")
	if f != nil {
		defer os.Remove(f.Name())
	}
	if err != nil {
		t.Fatal(err)
	}

	repoPath := repos.RepoPath(filepath.Dir(f.Name()))
	repoName := repos.RepoName(filepath.Dir(filepath.Base(f.Name())))
	repos.RegisterRepository(repoName, repoPath)

	doc := config.Document{
		Path: filepath.Base(f.Name()),
	}

	return ParseMarkdown(repoName, &doc)
}

// @llr REQ-TRAQ-SWL-2, REQ-TRAQ-SWL-3, REQ-TRAQ-SWL-4, REQ-TRAQ-SWL-5, REQ-TRAQ-SWL-83, REQ-TRAQ-SWL-84
func checkParseError(t *testing.T, content string, expectedError string) {
	_, _, err := doParse(t, content)
	if err == nil {
		t.Errorf("content `%s` does not generate error `%s`", content, expectedError)
	}
	assert.Contains(t, err.Error(), expectedError)
}

// @llr REQ-TRAQ-SWL-2, REQ-TRAQ-SWL-3, REQ-TRAQ-SWL-4, REQ-TRAQ-SWL-5, REQ-TRAQ-SWL-83, REQ-TRAQ-SWL-84
func checkParseOk(t *testing.T, content string, expectedFlow []*Flow, expectedReqs []*Req) {
	f, err := createTempFile(content, "checkParse")
	if f != nil {
		defer os.Remove(f.Name())
	}
	if err != nil {
		t.Fatal(err)
	}

	repoPath := repos.RepoPath(filepath.Dir(f.Name()))
	repoName := repos.RepoName(filepath.Dir(filepath.Base(f.Name())))
	repos.RegisterRepository(repoName, repoPath)

	doc := config.Document{
		Path: filepath.Base(f.Name()),
	}

	reqs, flow, err := ParseMarkdown(repoName, &doc)

	if err != nil {
		t.Errorf("content: `%s`\nshould not generate error: %v", content, err)
		return
	}

	for i := range reqs {
		// Set the document and repo name in the expected requirement
		expectedReqs[i].Document = &doc
		expectedReqs[i].RepoName = repoName

		if !reflect.DeepEqual(reqs[i], expectedReqs[i]) {
			t.Errorf("content: `%s`\nparsed into: %#v\ninstead of: %#v",
				content, reqs[i], expectedReqs[i])
		}
	}

	for i := range flow {
		// Set the document and repo name in the expected requirement
		expectedFlow[i].Document = &doc
		expectedFlow[i].RepoName = repoName

		if !reflect.DeepEqual(flow[i], expectedFlow[i]) {
			t.Errorf("content: `%s`\nparsed into: %#v\ninstead of: %#v",
				content, flow[i], expectedFlow[i])
		}
	}
}

// createTempFile creates a temporary file. It is the caller's responsibility
// to remove the file when not nil.
// @llr REQ-TRAQ-SWL-2, REQ-TRAQ-SWL-3, REQ-TRAQ-SWL-4, REQ-TRAQ-SWL-5
func createTempFile(content, prefix string) (*os.File, error) {
	f, err := ioutil.TempFile("", prefix)
	if err != nil {
		return nil, err
	}
	if _, err = f.WriteString(content); err != nil {
		return f, err
	}
	if err = f.Close(); err != nil {
		return f, err
	}
	return f, nil
}

// @llr REQ-TRAQ-SWL-3
func TestParseReq(t *testing.T) {
	r, err := parseReq(`REQ-TEST-SWL-1 title
body

body

###### Attributes:
- Rationale: This is why.
- Parents: REQ-TEST-SYS-1
- Attribute which will never exist: exists
`)
	assert.Nil(t, err)
	assert.Equal(t, "REQ-TEST-SWL-1", r.ID)
	assert.Equal(t, "title", r.Title)
	assert.Equal(t, "body\n\nbody\n", r.Body)
	assert.Equal(t, "This is why.", r.Attributes["RATIONALE"])
	assert.Equal(t, "exists", r.Attributes["ATTRIBUTE WHICH WILL NEVER EXIST"])
	assert.Equal(t, []string{"REQ-TEST-SYS-1"}, r.ParentIds)
}

// @llr REQ-TRAQ-SWL-3
func TestParseReq_Empty(t *testing.T) {
	_, err := parseReq(`REQ-TEST-SWL-1 title

`)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "Requirement must not be empty: REQ-TEST-SWL-1")
}

// @llr REQ-TRAQ-SWL-3
func TestParseReq_Deleted(t *testing.T) {
	// Make sure it can be parsed even when it has no description.
	r, err := parseReq(`REQ-T-SYS-1 DELETED`)
	assert.Nil(t, err)
	assert.True(t, r.IsDeleted())

	// Make sure it can be parsed when it has description.
	r, err = parseReq(`REQ-TEST-SWL-1 DELETED Some title
body

###### Attributes:
- Rationale: This is why.
- Parents: REQ-TEST-SYS-1
`)
	assert.Nil(t, err)
	assert.Equal(t, "REQ-TEST-SWL-1", r.ID)
	assert.Equal(t, "DELETED Some title", r.Title)
	assert.Equal(t, "body\n", r.Body)
	assert.Equal(t, "This is why.", r.Attributes["RATIONALE"])
	assert.Equal(t, []string{"REQ-TEST-SYS-1"}, r.ParentIds)
	assert.True(t, r.IsDeleted())
}

// @llr REQ-TRAQ-SWL-3
func TestParseReq_EmptyBody(t *testing.T) {
	_, err := parseReq(`REQ-TEST-SWL-1 title

## Attributes:
- A: B
`)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "Requirement body must not be empty: REQ-TEST-SWL-1")
}

// @llr REQ-TRAQ-SWL-3
func TestParseReq_FlexibleAttributesHeading(t *testing.T) {
	r, err := parseReq(`REQ-TEST-SWL-1 title
body
## Attributes:
- Rationale: This is why.
`)
	assert.Nil(t, err)
	assert.Equal(t, "This is why.", r.Attributes["RATIONALE"])
}

// @llr REQ-TRAQ-SWL-3
func TestParseReq_NoAttributes(t *testing.T) {
	r, err := parseReq(`REQ-TEST-SWL-1 title
body`)
	assert.Nil(t, err)
	assert.Equal(t, "body", r.Body)
}

// @llr REQ-TRAQ-SWL-3
func TestParseReq_EmptyAttributesSection(t *testing.T) {
	_, err := parseReq(`REQ-TEST-SWL-1 title
body
###### Attributes:
`)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Requirement REQ-TEST-SWL-1 contains an attribute section but no attributes")
}

// @llr REQ-TRAQ-SWL-3
func TestParseReq_DuplicateAttributes(t *testing.T) {
	_, err := parseReq(`REQ-TEST-SWL-1 title
body
## Attributes:
- Rationale: This is why.
- Rationale: This is why.
`)
	assert.EqualError(t, err, `requirement REQ-TEST-SWL-1 contains duplicate attribute: "RATIONALE"`)
}

// @llr REQ-TRAQ-SWL-3
func TestParseReq_Parents(t *testing.T) {
	r, err := parseReq(`REQ-T-SWL-1 title
body
## Attributes:
- Parent: REQ-T-SWH-1, REQ-T-SWH-1000 REQ-T-SWH-1001
`)
	assert.Nil(t, err)
	assert.Equal(t, []string{"REQ-T-SWH-1", "REQ-T-SWH-1000", "REQ-T-SWH-1001"}, r.ParentIds)
}

// @llr REQ-TRAQ-SWL-3
func TestParseReq_InvalidParents(t *testing.T) {
	_, err := parseReq(`REQ-TEST-SWL-1 title
body
## Attributes:
- Parents: REQ-TEST-SWH-1 and REQ-TEST-SWH-2
`)
	assert.EqualError(t, err, `requirement REQ-TEST-SWL-1 parents: unparseable as list of requirement ids: " and " in "REQ-TEST-SWH-1 and REQ-TEST-SWH-2"`)
}

// @llr REQ-TRAQ-SWL-3
func TestParseReq_InvalidParents2(t *testing.T) {
	_, err := parseReq(`REQ-TEST-SWL-1 title
body
## Attributes:
- Parents: TODO
`)
	assert.EqualError(t, err, `requirement REQ-TEST-SWL-1 parents: unparseable as list of requirement ids: "TODO"`)
}

// @llr REQ-TRAQ-SWL-3
func TestParseReq_InvalidParents3(t *testing.T) {
	_, err := parseReq(`REQ-TEST-SWL-1 title
body
## Attributes:
- Parents: REQ-VXS-SYS-123, TODO
`)
	assert.EqualError(t, err, `requirement REQ-TEST-SWL-1 parents: unparseable as list of requirement ids: ", TODO" in "REQ-VXS-SYS-123, TODO"`)
}

// @llr REQ-TRAQ-SWL-3
func TestParseReq_InvalidParents4(t *testing.T) {
	_, err := parseReq(`REQ-TEST-SWL-1 title
body
## Attributes:
- Parents: REQ-VXS-SYS-123, REQ-VXS-456
`)
	assert.EqualError(t, err, `requirement REQ-TEST-SWL-1 parents: unparseable as list of requirement ids: ", REQ-VXS-456" in "REQ-VXS-SYS-123, REQ-VXS-456"`)
}

// @llr REQ-TRAQ-SWL-5
func TestParseReqTable(t *testing.T) {
	tableOffset := 127
	reqs, err := parseReqTable(`| ID | Title | Body | Rationale | Verification | Safety impact | Parents |
| ----- | ----- | ----- | ----- | ----- | ----- |
| REQ-TEST-SYS-1 | Section 1 | Body of requirement 1. | Rationale 1 | Test 1 | Impact 1 | |
| REQ-TEST-SYS-2 | Section 2 | Body of requirement 2. | Rationale 2 | Test 2 | Impact 2 | |
| REQ-TEST-SYS-3 | Section 3 | Body of requirement 3. | Rationale 3 | Test 3 | Impact 3 | REQ-TEST-SYS-1 |
| REQ-TEST-SYS-4 | Section 4 | Body of requirement 4. | Rationale 4 | Test 4 | Impact 4 | REQ-TEST-SYS-1, REQ-TEST-SYS-2 |`, tableOffset, nil)

	assert.Nil(t, err)
	assert.Equal(t, 4, len(reqs))

	for i, r := range reqs {
		assert.Equal(t, fmt.Sprintf("REQ-TEST-SYS-%d", i+1), r.ID)
		assert.Equal(t, fmt.Sprintf("Section %d", i+1), r.Title)
		assert.Equal(t, fmt.Sprintf("Body of requirement %d.", i+1), r.Body)
		assert.Equal(t, fmt.Sprintf("Rationale %d", i+1), r.Attributes["RATIONALE"])
		assert.Equal(t, fmt.Sprintf("Test %d", i+1), r.Attributes["VERIFICATION"])
		assert.Equal(t, fmt.Sprintf("Impact %d", i+1), r.Attributes["SAFETY IMPACT"])
		assert.Equal(t, tableOffset+i+2, r.Position)
	}
}

// @llr REQ-TRAQ-SWL-5
func TestParseReqTable_NoIDCol(t *testing.T) {
	_, err := parseReqTable(`| Title | Body | Rationale | Verification | Safety impact |
| ----- | ----- | ----- | ----- | ----- |
| Section 1 | Body of requirement 1. | Rationale 1 | Test 1 | Impact 1 |`, 0, nil)

	assert.EqualError(t, err, "requirement table must have at least 2 columns, first column head must be \"ID\"")
}

// @llr REQ-TRAQ-SWL-5
func TestParseReqTable_OneCol(t *testing.T) {
	_, err := parseReqTable(`| ID |
| ----- |
| REQ-TEST-SYS-1 |`, 0, nil)

	assert.EqualError(t, err, "requirement table must have at least 2 columns, first column head must be \"ID\"")
}

// @llr REQ-TRAQ-SWL-5
func TestParseReqTable_MissingCell(t *testing.T) {
	_, err := parseReqTable(`| ID | Title | Body | Rationale | Verification | Safety impact |
| ----- | ----- | ----- | ----- | ----- | ----- |
| REQ-TEST-SYS-1 | Section 1 | Body of requirement 1. | Rationale 1 | Test 1 |`, 0, nil)

	assert.EqualError(t, err, "too few cells on row 3 of requirement table")
}

// @llr REQ-TRAQ-SWL-5
func TestParseReqTable_BadID(t *testing.T) {
	_, err := parseReqTable(`| ID | Title | Body | Rationale | Verification | Safety impact |
| ----- | ----- | ----- | ----- | ----- | ----- |
| REQ-TEST-1 | Section 1 | Body of requirement 1. | Rationale 1 | Test 1 | Impact 1 |`, 0, nil)

	assert.EqualError(t, err, "malformed requirement: found only malformed ID: \"REQ-TEST-1\" (doesn't match \"(REQ|ASM)-(\\\\w+)-(\\\\w+)-(\\\\d+)\")")
}

// @llr REQ-TRAQ-SWL-5
func TestParseReqTable_MissingID(t *testing.T) {
	_, err := parseReqTable(`| ID | Title | Body | Rationale | Verification | Safety impact |
| ----- | ----- | ----- | ----- | ----- | ----- |
|  | Section 1 | Body of requirement 1. | Rationale 1 | Test 1 | Impact 1 |`, 0, nil)

	assert.EqualError(t, err, "malformed requirement: missing ID in first 40 characters: \"\"")
}
