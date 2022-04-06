package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestParseMarkdown checks that parseMarkdown finds the requirements blocks
// correctly.
func TestParseMarkdown(t *testing.T) {

	// Heading style requirements
	checkParse(t, `
# Title
#### REQ-TEST-SYS-5 My First Requirement
##### Heading part of a req
#### REQ-TEST-SYS-6
Content mentioning REQ-TEST-SYS-1
REQ-TEST-SYS-2
### Title2
#### REQ-TEST-SYS-7 My Last Requirement
Some more content
`,
		"",
		&Req{ID: "REQ-TEST-SYS-5",
			IDNumber:   5,
			Title:      "My First Requirement",
			Body:       "##### Heading part of a req",
			Attributes: map[string]string{}},
		&Req{ID: "REQ-TEST-SYS-6",
			IDNumber:   6,
			Title:      "Content mentioning REQ-TEST-SYS-1",
			Body:       "REQ-TEST-SYS-2",
			Attributes: map[string]string{}},
		&Req{ID: "REQ-TEST-SYS-7",
			IDNumber:   7,
			Title:      "My Last Requirement",
			Body:       "Some more content",
			Attributes: map[string]string{}})

	checkParse(t, `# REQ-TEST-SYS-5 REQ-TEST-SYS-6`, `malformed requirement title: too many IDs on line 1:`)
	checkParse(t, `
# REQ-TEST-SYS-5
## REQ-TEST-SYS-6`,
		"requirement heading on line 3 must be at same level as requirement heading on line 2 (2 != 1):")
	checkParse(t, `
## REQ-TEST-SYS-5
# REQ-TEST-SYS-6`,
		"requirement heading on line 3 must be at same level as requirement heading on line 2 (1 != 2):")
	checkParse(t, `
# REQ-TEST-SYS-5
# Title`,
		"non-requirement heading on line 3 at same level as requirement heading on line 2 (1):")
	checkParse(t, `
# Title
# REQ-TEST-SYS-5`,
		"requirement heading on line 3 at same level as previous heading on line 2 (1):")

	// Table style requirements
	checkParse(t, `
| ID | Title | Body |
| REQ-TEST-SYS-5 | My First Requirement | Heading part of a req |
| REQ-TEST-SYS-6 | Content mentioning REQ-TEST-SYS-1 | REQ-TEST-SYS-2 |
| REQ-TEST-SYS-7 | My Last Requirement | Some more content |
`,
		"",
		&Req{ID: "REQ-TEST-SYS-5",
			IDNumber:   5,
			Title:      "My First Requirement",
			Body:       "Heading part of a req",
			Attributes: map[string]string{}},
		&Req{ID: "REQ-TEST-SYS-6",
			IDNumber:   6,
			Title:      "Content mentioning REQ-TEST-SYS-1",
			Body:       "REQ-TEST-SYS-2",
			Attributes: map[string]string{}},
		&Req{ID: "REQ-TEST-SYS-7",
			IDNumber:   7,
			Title:      "My Last Requirement",
			Body:       "Some more content",
			Attributes: map[string]string{}})

	// Mixed style requirements
	checkParse(t, `
# Title
#### REQ-TEST-SYS-5 My First Requirement
##### Heading part of a req
| ID | Title | Body |
| REQ-TEST-SYS-6 | Content mentioning REQ-TEST-SYS-1 | REQ-TEST-SYS-2 |
| REQ-TEST-SYS-7 | My Last Requirement | Some more content |
`,
		"",
		&Req{ID: "REQ-TEST-SYS-5",
			IDNumber:   5,
			Title:      "My First Requirement",
			Body:       "##### Heading part of a req",
			Attributes: map[string]string{}},
		&Req{ID: "REQ-TEST-SYS-6",
			IDNumber:   6,
			Title:      "Content mentioning REQ-TEST-SYS-1",
			Body:       "REQ-TEST-SYS-2",
			Attributes: map[string]string{}},
		&Req{ID: "REQ-TEST-SYS-7",
			IDNumber:   7,
			Title:      "My Last Requirement",
			Body:       "Some more content",
			Attributes: map[string]string{}})
}

func checkParse(t *testing.T, content, expectedError string, expectedReqs ...*Req) {
	f, err := createTempFile(content, "checkParse")
	if f != nil {
		defer os.Remove(f.Name())
	}
	if err != nil {
		t.Fatal(err)
	}
	reqs, err := parseMarkdown(f.Name())
	if expectedError == "" {
		if err != nil {
			t.Errorf("content: `%s`\nshould not generate error: %v", content, err)
		} else {
			for i, _ := range reqs {
				if !reflect.DeepEqual(reqs[i], expectedReqs[i]) {
					t.Errorf("content: `%s`\nparsed into: %#v\ninstead of: %#v",
						content, reqs[i], expectedReqs[i])
				}
			}
		}
	} else {
		if err == nil {
			t.Errorf("content `%s` does not generate error `%s`", content, expectedError)
		}
		assert.Contains(t, err.Error(), expectedError)
	}
}

// createTempFile creates a temporary file. It is the caller's responsibility
// to remove the file when not nil.
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

func TestParseReq(t *testing.T) {
	r, err := parseReq(`REQ-TEST-SWL-1 title
body

body

###### Attributes:
- Rationale: This is why.
- Parents: REQ-TEST-SYS-1.
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

func TestParseReq_InvalidType(t *testing.T) {
	_, err := parseReq(`REQ-TEST-WILLNEVEREXIST-1 title
body
`)
	assert.EqualError(t, err, `Invalid request type: "WILLNEVEREXIST"`)
}

func TestParseReq_Empty(t *testing.T) {
	_, err := parseReq(`REQ-TEST-SWL-1 title

`)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "Requirement must not be empty: REQ-TEST-SWL-1")
}

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

func TestParseReq_EmptyBody(t *testing.T) {
	_, err := parseReq(`REQ-TEST-SWL-1 title

## Attributes:
- A: B
`)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "Requirement body must not be empty: REQ-TEST-SWL-1")
}

func TestParseReq_FlexibleAttributesHeading(t *testing.T) {
	r, err := parseReq(`REQ-TEST-SWL-1 title
body
## Attributes:
- Rationale: This is why.
`)
	assert.Nil(t, err)
	assert.Equal(t, "This is why.", r.Attributes["RATIONALE"])
}

func TestParseReq_NoAttributes(t *testing.T) {
	r, err := parseReq(`REQ-TEST-SWL-1 title
body`)
	assert.Nil(t, err)
	assert.Equal(t, "body", r.Body)
}

func TestParseReq_EmptyAttributesSection(t *testing.T) {
	_, err := parseReq(`REQ-TEST-SWL-1 title
body
###### Attributes:
`)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Requirement REQ-TEST-SWL-1 contains an attribute section but no attributes")
}

func TestParseReq_DuplicateAttributes(t *testing.T) {
	_, err := parseReq(`REQ-TEST-SWL-1 title
body
## Attributes:
- Rationale: This is why.
- Rationale: This is why.
`)
	assert.EqualError(t, err, `requirement REQ-TEST-SWL-1 contains duplicate attribute: "RATIONALE"`)
}

func TestParseReq_Parents(t *testing.T) {
	r, err := parseReq(`REQ-T-SWL-1 title
body
## Attributes:
- Parent: REQ-T-SWH-1, REQ-T-SWH-1000 REQ-T-SWH-1001
`)
	assert.Nil(t, err)
	assert.Equal(t, []string{"REQ-T-SWH-1", "REQ-T-SWH-1000", "REQ-T-SWH-1001"}, r.ParentIds)
}

func TestParseReq_InvalidParents(t *testing.T) {
	_, err := parseReq(`REQ-TEST-SWL-1 title
body
## Attributes:
- Parents: REQ-TEST-SWH-1 and REQ-TEST-SWH-2
`)
	assert.EqualError(t, err, `requirement REQ-TEST-SWL-1 parents: unparseable as list of requirement ids: " and " in "REQ-TEST-SWH-1 and REQ-TEST-SWH-2"`)
}

func TestParseReqTable(t *testing.T) {
	reqs, err := parseReqTable(`| ID | Title | Body | Rationale | Verification | Safety impact | Parents |
| ----- | ----- | ----- | ----- | ----- | ----- |
| REQ-TEST-SYS-1 | Section 1 | Body of requirement 1. | Rationale 1 | Test 1 | Impact 1 | |
| REQ-TEST-SYS-2 | Section 2 | Body of requirement 2. | Rationale 2 | Test 2 | Impact 2 | |
| REQ-TEST-SYS-3 | Section 3 | Body of requirement 3. | Rationale 3 | Test 3 | Impact 3 | REQ-TEST-SYS-1 |
| REQ-TEST-SYS-4 | Section 4 | Body of requirement 4. | Rationale 4 | Test 4 | Impact 4 | REQ-TEST-SYS-1, REQ-TEST-SYS-2 |`, nil)

	assert.Nil(t, err)
	assert.Equal(t, 4, len(reqs))

	for i, r := range reqs {
		assert.Equal(t, fmt.Sprintf("REQ-TEST-SYS-%d", i+1), r.ID)
		assert.Equal(t, fmt.Sprintf("Section %d", i+1), r.Title)
		assert.Equal(t, fmt.Sprintf("Body of requirement %d.", i+1), r.Body)
		assert.Equal(t, fmt.Sprintf("Rationale %d", i+1), r.Attributes["RATIONALE"])
		assert.Equal(t, fmt.Sprintf("Test %d", i+1), r.Attributes["VERIFICATION"])
		assert.Equal(t, fmt.Sprintf("Impact %d", i+1), r.Attributes["SAFETY IMPACT"])
	}
}

func TestParseReqTable_NoIDCol(t *testing.T) {
	_, err := parseReqTable(`| Title | Body | Rationale | Verification | Safety impact |
| ----- | ----- | ----- | ----- | ----- |
| Section 1 | Body of requirement 1. | Rationale 1 | Test 1 | Impact 1 |`, nil)

	assert.EqualError(t, err, "requirement table must have at least 2 columns, first column head must be \"ID\"")
}

func TestParseReqTable_OneCol(t *testing.T) {
	_, err := parseReqTable(`| ID |
| ----- |
| REQ-TEST-SYS-1 |`, nil)

	assert.EqualError(t, err, "requirement table must have at least 2 columns, first column head must be \"ID\"")
}

func TestParseReqTable_MissingCell(t *testing.T) {
	_, err := parseReqTable(`| ID | Title | Body | Rationale | Verification | Safety impact |
| ----- | ----- | ----- | ----- | ----- | ----- |
| REQ-TEST-SYS-1 | Section 1 | Body of requirement 1. | Rationale 1 | Test 1 |`, nil)

	assert.EqualError(t, err, "too few cells on row 3 of requirement table")
}

func TestParseReqTable_BadID(t *testing.T) {
	_, err := parseReqTable(`| ID | Title | Body | Rationale | Verification | Safety impact |
| ----- | ----- | ----- | ----- | ----- | ----- |
| REQ-TEST-1 | Section 1 | Body of requirement 1. | Rationale 1 | Test 1 | Impact 1 |`, nil)

	assert.EqualError(t, err, "malformed requirement: found only malformed ID: \"REQ-TEST-1\" (doesn't match \"REQ-(\\\\w+)-(\\\\w+)-(\\\\d+)\")")
}

func TestParseReqTable_MissingID(t *testing.T) {
	_, err := parseReqTable(`| ID | Title | Body | Rationale | Verification | Safety impact |
| ----- | ----- | ----- | ----- | ----- | ----- |
|  | Section 1 | Body of requirement 1. | Rationale 1 | Test 1 | Impact 1 |`, nil)

	assert.EqualError(t, err, "malformed requirement: missing ID in first 40 characters: \"\"")
}
