package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseReq(t *testing.T) {
	r, err := ParseReq(`REQ-TEST-SWL-1 title
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
	_, err := ParseReq(`REQ-TEST-WILLNEVEREXIST-1 title
body
`)
	assert.EqualError(t, err, `Invalid request type: "WILLNEVEREXIST"`)
}

func TestParseReq_Empty(t *testing.T) {
	_, err := ParseReq(`REQ-TEST-SWL-1 title

`)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "Requirement must not be empty: REQ-TEST-SWL-1")
}

func TestParseReq_Deleted(t *testing.T) {
	// Make sure it can be parsed even when it has no description.
	r, err := ParseReq(`REQ-T-SYS-1 DELETED`)
	assert.Nil(t, err)
	assert.True(t, r.IsDeleted())

	// Make sure it can be parsed when it has description.
	r, err = ParseReq(`REQ-TEST-SWL-1 DELETED Some title
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
	_, err := ParseReq(`REQ-TEST-SWL-1 title

## Attributes:
- A: B
`)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "Requirement body must not be empty: REQ-TEST-SWL-1")
}

func TestParseReq_FlexibleAttributesHeading(t *testing.T) {
	r, err := ParseReq(`REQ-TEST-SWL-1 title
body
## Attributes:
- Rationale: This is why.
`)
	assert.Nil(t, err)
	assert.Equal(t, "This is why.", r.Attributes["RATIONALE"])
}

func TestParseReq_NoAttributes(t *testing.T) {
	r, err := ParseReq(`REQ-TEST-SWL-1 title
body`)
	assert.Nil(t, err)
	assert.Equal(t, "body", r.Body)
}

func TestParseReq_EmptyAttributesSection(t *testing.T) {
	_, err := ParseReq(`REQ-TEST-SWL-1 title
body
###### Attributes:
`)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Requirement REQ-TEST-SWL-1 contains an attribute section but no attributes")
}

func TestParseReq_DuplicateAttributes(t *testing.T) {
	_, err := ParseReq(`REQ-TEST-SWL-1 title
body
## Attributes:
- Rationale: This is why.
- Rationale: This is why.
`)
	assert.EqualError(t, err, `requirement REQ-TEST-SWL-1 contains duplicate attribute: "RATIONALE"`)
}

func TestParseReq_Parents(t *testing.T) {
	r, err := ParseReq(`REQ-T-SWL-1 title
body
## Attributes:
- Parent: REQ-T-SWH-1, REQ-T-SWH-1000 REQ-T-SWH-1001
`)
	assert.Nil(t, err)
	assert.Equal(t, []string{"REQ-T-SWH-1", "REQ-T-SWH-1000", "REQ-T-SWH-1001"}, r.ParentIds)
}

func TestParseReq_InvalidParents(t *testing.T) {
	_, err := ParseReq(`REQ-TEST-SWL-1 title
body
## Attributes:
- Parents: REQ-TEST-SWH-1 and REQ-TEST-SWH-2
`)
	assert.EqualError(t, err, `requirement REQ-TEST-SWL-1 parents: unparseable as list of requirement ids: " and " in "REQ-TEST-SWH-1 and REQ-TEST-SWH-2"`)
}
