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
#### REQ-TEST-SYS-5
##### Heading part of a req
#### REQ-TEST-SYS-6
Content mentioning REQ-TEST-SYS-1
REQ-TEST-SYS-2
### Title2
#### REQ-TEST-SYS-7
`,
		"",
		"REQ-TEST-SYS-5\n##### Heading part of a req\n",
		"REQ-TEST-SYS-6\nContent mentioning REQ-TEST-SYS-1\nREQ-TEST-SYS-2\n",
		"REQ-TEST-SYS-7\n")

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
