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
`)
	assert.Nil(t, err)
	assert.Equal(t, "REQ-TEST-SWL-1", r.ID)
	assert.Equal(t, "title", r.Title)
	assert.Equal(t, "<p>body</p>\n<p>body</p>\n", string(r.Body))
	assert.Equal(t, "This is why.", r.Attributes["RATIONALE"])
	assert.Equal(t, []string{"REQ-TEST-SYS-1"}, r.ParentIds)
}

func TestParseReq_NoAttributes(t *testing.T) {
	r, err := ParseReq(`REQ-TEST-SWL-1 title
body`)
	assert.Nil(t, err)
	assert.Equal(t, "<p>body</p>\n", string(r.Body))
}

func TestParseReq_EmptyAttributesSection(t *testing.T) {
	_, err := ParseReq(`REQ-TEST-SWL-1 title
body
###### Attributes:
`)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Requirement REQ-TEST-SWL-1 contains an attribute section but no attributes")
}
