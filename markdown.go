package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
)

var (
	// For detecting ATX Headings, see http://spec.commonmark.org/0.27/#atx-headings
	reATXHeading = regexp.MustCompile(`(?m)^ {0,3}(#{1,6})( +(.*)( #* *)?)?$`)
)

// ParseMarkdown parses a certification document and returns the found
// requirements.
func ParseMarkdown(f string) ([]string, error) {
	var (
		reqs []string

		lastHeadingLevel int // The level of the last ATX heading.
		lastHeadingLine  int // The line number of the last ATX heading.
		inReq            bool
		reqLevel         int // The level of the ATX heading starting the requirement.
		reqLine          int // The line number of the ATX heading starting the requirement.
		reqBuf           bytes.Buffer
	)

	r, err := os.Open(f)
	if err != nil {
		return nil, err
	}
	scan := bufio.NewScanner(r)

	for lno := 1; scan.Scan(); lno++ {
		line := scan.Text()

		var level int
		parts := reATXHeading.FindStringSubmatch(line)
		if parts != nil {
			// An ATX heading.
			level = len(parts[1])
			title := parts[3]
			reqIDs := ReReqID.FindAllString(title, -1)
			if len(reqIDs) > 1 {
				return nil, fmt.Errorf("malformed requirement title: too many IDs on line %d: %q", lno, line)
			}
			headingHasReqID := len(reqIDs) == 1
			// Figure out what to do with this heading.
			end := false
			start := false
			if inReq {
				// A request is currently being parsed.
				if headingHasReqID {
					// This is a requirement heading.
					// The level must be the same as the current requirement.
					if level != reqLevel {
						return nil, fmt.Errorf("requirement heading on line %d must be at same level as requirement heading on line %d (%d != %d): %q", lno, reqLine, level, reqLevel, line)
					}
					// Besides starting a requirement, this heading also ends the current one.
					end = true
					start = true
				} else {
					// No requirement ID on this heading.
					// The heading level must be lower or higher than the current
					// requirement's heading level. We don't want to mix requirements
					// with other headings of the same level, in the same section.
					if level == reqLevel {
						return nil, fmt.Errorf("non-requirement heading on line %d at same level as requirement heading on line %d (%d): %q", lno, reqLine, level, line)
					}
					if level < reqLevel {
						// Higher-level heading.
						end = true
					} else {
						// Lower-level heading. Will be included in the current requirement.
					}
				}
			} else {
				// Nothing going on yet.
				if headingHasReqID {
					// Can be the first one or the first one in another section.
					if level == lastHeadingLevel {
						return nil, fmt.Errorf("requirement heading on line %d at same level as previous heading on line %d (%d): %q", lno, lastHeadingLine, level, line)
					}
					start = true
				} else {
					// No requirement ID on this heading. Will be ignored.
				}
			}

			if end {
				// Close the current requirement.
				reqs = append(reqs, reqBuf.String())
				inReq = false
			}
			if start {
				// Start a new requirement at the current line.
				inReq = true
				reqLevel = level
				reqLine = lno
				reqBuf.Reset()
				line = title
			}
		}
		if inReq {
			reqBuf.WriteString(line)
			reqBuf.WriteString("\n")
		}
		if level > 0 {
			lastHeadingLevel = level
			lastHeadingLine = lno
		}
	}
	if err := scan.Err(); err != nil {
		return nil, err
	}

	if inReq {
		// Close the current requirement, we're at the end.
		reqs = append(reqs, reqBuf.String())
	}

	return reqs, nil
}
