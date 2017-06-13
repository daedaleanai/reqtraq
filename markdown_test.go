package main

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestParseMarkdown checks that ParseMarkdown finds the requirements blocks
// correctly.
func TestParseMarkdown(t *testing.T) {
	checkParse(t, `
# Title
#### REQ-0-TEST-SYS-005
##### Heading part of a req
#### REQ-0-TEST-SYS-006
Content mentioning REQ-0-TEST-SYS-001
REQ-0-TEST-SYS-002
### Title2
#### REQ-0-TEST-SYS-007
`,
		"",
		"REQ-0-TEST-SYS-005\n##### Heading part of a req\n",
		"REQ-0-TEST-SYS-006\nContent mentioning REQ-0-TEST-SYS-001\nREQ-0-TEST-SYS-002\n",
		"REQ-0-TEST-SYS-007\n")

	checkParse(t, `# REQ-0-TEST-SYS-005 REQ-0-TEST-SYS-006`, `malformed requirement title: too many IDs on line 1:`)
	checkParse(t, `
# REQ-0-TEST-SYS-005
## REQ-0-TEST-SYS-006`,
		"requirement heading on line 3 must be at same level as requirement heading on line 2 (2 != 1):")
	checkParse(t, `
## REQ-0-TEST-SYS-005
# REQ-0-TEST-SYS-006`,
		"requirement heading on line 3 must be at same level as requirement heading on line 2 (1 != 2):")
	checkParse(t, `
# REQ-0-TEST-SYS-005
# Title`,
		"non-requirement heading on line 3 at same level as requirement heading on line 2 (1):")
	checkParse(t, `
# Title
# REQ-0-TEST-SYS-005`,
		"requirement heading on line 3 at same level as previous heading on line 2 (1):")
}

func checkParse(t *testing.T, content, expectedError string, expectedReqs ...string) {
	f, err := createTempFile(content, "checkParse")
	if f != nil {
		defer os.Remove(f.Name())
	}
	if err != nil {
		t.Fatal(err)
	}
	reqs, err := ParseMarkdown(f.Name())
	if expectedError == "" {
		if err != nil {
			t.Errorf("content: `%s`\nshould not generate error: %v", content, expectedError)
		} else {
			if !reflect.DeepEqual(reqs, expectedReqs) {
				t.Errorf("content: `%s`\nparsed into: %v\ninstead of: %v", content, reqs, expectedReqs)
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
