package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/daedaleanai/reqtraq/config"
)

var (
	// For detecting ATX Headings, see http://spec.commonmark.org/0.27/#atx-headings
	reATXHeading = regexp.MustCompile(`(?m)^ {0,3}(#{1,6})( +(.*)( #* *)?)?$`)
	// REQ, project number, project abbreviation, req type, req number
	// For example: REQ-TRAQ-SWH-4
	reReqIdStr                 = `REQ-(\w+)-(\w+)-(\d+)`
	ReReqID                    = regexp.MustCompile(reReqIdStr)
	reReqIDBad                 = regexp.MustCompile(`(?i)REQ-((\d+)|((\w+)-(\d+)))`)
	ReReqDeleted               = regexp.MustCompile(reReqIdStr + ` DELETED`)
	reAttributesSectionHeading = regexp.MustCompile(`(?m)\n#{2,6} Attributes:$`)
	reReqKWD                   = regexp.MustCompile(`(?i)- ([^:]+): `)
)

// ParseCertdoc checks the f filename is a valid certdoc name then parses raw requirements out of it.
// @llr REQ-TRAQ-SWL-20
func ParseCertdoc(filename string) ([]*Req, error) {
	ext := path.Ext(filename)
	if strings.ToLower(ext) != ".md" {
		return nil, fmt.Errorf("Invalid extension: '%s'. Only '.md' is supported", strings.ToLower(ext))
	}

	basename := strings.TrimSuffix(path.Base(filename), ext)
	// check if the structure of the filename is correct
	parts := reCertdoc.FindStringSubmatch(basename)
	if parts == nil {
		return nil, fmt.Errorf("Invalid file name: '%s'. Certification doc file name must match %v",
			basename, reCertdoc)
	}

	// check the document type code
	docType := parts[3]
	correctNumber, ok := config.DocTypeToDocId[docType]
	if !ok {
		return nil, fmt.Errorf("Invalid document type: '%s'. Must be one of %v",
			docType, config.DocTypeToDocId)
	}

	// check the document type number
	docNumber := parts[2]
	if correctNumber != docNumber {
		return nil, fmt.Errorf("Document number for type '%s' must be '%s', and not '%s'",
			docType, correctNumber, docNumber)
	}
	return parseMarkdown(filename)
}

// parseMarkdown parses a certification document and returns the found
// requirements.
// @llr REQ-TRAQ-SWL-21
func parseMarkdown(f string) ([]*Req, error) {
	var (
		reqs []*Req

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
				newReq, err := parseReq(reqBuf.String())
				if err != nil {
					return nil, err
				}
				reqs = append(reqs, newReq)
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
		newReq, err := parseReq(reqBuf.String())
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, newReq)
	}

	return reqs, nil
}

// parseReq finds the first REQ-XXX tag and the reserved words and distills a Req from it.
//
// parseReq parses according to the 'soft' format defined in the SRS:
//            REQ-ID (text)
//            [Rationale:....]
//            [Parent[s]: REQ-ID[, REQ-ID...]]
//            [Safety Impact:...]
//            [Verification:...]
//            [Urgent:...]
//            [Important:...]
//            [Mode:...]
//            [Provenance:...]
//
// parseReq does NOT validate the values or check if the mandatory attributes are set; use
// the Req.Check() method for that.
//
// Since the parsing is rather 'soft', ParseReq returns verbose errors indicating problems in
// a helpful way, meaning they at least provide enough context for the user to find the text.
//
// @llr REQ-TRAQ-SWL-13
func parseReq(txt string) (*Req, error) {
	head := txt
	if len(head) > 40 {
		head = head[:40]
	}
	defid := ReReqID.FindStringSubmatchIndex(txt)
	if len(defid) == 0 {
		if reReqIDBad.MatchString(head) {
			return nil, fmt.Errorf("malformed requirement: found only malformed ID: %q (doesn't match %q)", head, ReReqID)
		}
		return nil, fmt.Errorf("malformed requirement: missing ID in first 40 characters: %q", head)
	}

	if defid[0] > 0 {
		return nil, fmt.Errorf("malformed requirement: ID must be at the start of the title: %q", head)
	}

	IDNumber, err := strconv.Atoi(txt[defid[6]:defid[7]])
	if err != nil {
		return nil, err
	}

	r := &Req{
		ID:         txt[defid[0]:defid[1]],
		IDNumber:   IDNumber,
		Attributes: map[string]string{},
	}

	var ok bool
	if r.Level, ok = config.ReqTypeToReqLevel[r.ReqType()]; !ok {
		return nil, fmt.Errorf("Invalid request type: %q", r.ReqType())
	}

	// chop defining ID and any punctuation
	txt = strings.TrimLeftFunc(txt[defid[1]:], IsPunctOrSpace)

	// The first line is the title.
	parts := strings.SplitN(strings.TrimSpace(txt), "\n", 2)
	r.Title = parts[0]

	if len(parts) < 2 {
		if r.IsDeleted() {
			// This is a placeholder for an obsolete requirement.
			return r, nil
		}
		// The definition of the non-deleted requirement has a single line,
		// so it has no description (body, attributes).
		return nil, fmt.Errorf("Requirement must not be empty: %s", r.ID)
	}

	// Next is the body, until the attributes section.
	bodyAndAttributes := parts[1]
	var attributesStart = len(bodyAndAttributes)
	ii := reAttributesSectionHeading.FindStringIndex(bodyAndAttributes)
	if ii != nil {
		attributesStart = ii[0]
		attributes := bodyAndAttributes[attributesStart:]
		kwdMatches := reReqKWD.FindAllStringSubmatchIndex(attributes, -1)
		if len(kwdMatches) == 0 {
			return nil, fmt.Errorf("Requirement %s contains an attribute section but no attributes", r.ID)
		}
		for i, v := range kwdMatches {
			key := strings.ToUpper(attributes[v[2]:v[3]])
			if key == "PARENT" { // make our lives easier, accept both, output only PARENTS
				key = "PARENTS"
			}
			e := len(attributes)
			if i < len(kwdMatches)-1 {
				e = kwdMatches[i+1][0]
			}
			if _, ok := r.Attributes[key]; ok {
				return nil, fmt.Errorf("requirement %s contains duplicate attribute: %q", r.ID, key)
			}
			r.Attributes[key] = strings.TrimSpace(attributes[v[1]:e])
		}
	}

	r.Body = bodyAndAttributes[:attributesStart]

	if strings.TrimSpace(r.Body) == "" {
		return nil, fmt.Errorf("Requirement body must not be empty: %s", r.ID)
	}

	// PARENTS must be punctuation/space separated list of parseable req-ids.
	parents := r.Attributes["PARENTS"]
	parmatch := ReReqID.FindAllStringSubmatchIndex(parents, -1)
	for i, ids := range parmatch {
		val := parents[ids[0]:ids[1]]
		r.ParentIds = append(r.ParentIds, val)
		if i > 0 {
			sep := parents[parmatch[i-1][1]:ids[0]]
			if strings.TrimFunc(sep, IsPunctOrSpace) != "" {
				return nil, fmt.Errorf("requirement %s parents: unparseable as list of requirement ids: %q in %q", r.ID, sep, parents)
			}
		}
	}

	return r, nil
}

func IsPunctOrSpace(r rune) bool {
	return unicode.IsSpace(r) || unicode.IsPunct(r)
}
