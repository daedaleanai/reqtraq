/*
Functions within this file are concerned with parsing requirements out of markdown documents.

The entry point is ParseCertdoc which in turns calls other functions as follows:
- ParseCertdoc: Checks filename is valid then calls:
- parseMarkdown: Scans file one line at a time looking for requirements that either formatted within ATX headings
                 or held in tables. For each ATX requirement or table calls:
- parseMarkdownFragment: Depending on the type of requirement calls one of the following functions.
- parseReq: Parses ATX heading requirements into the Req structure and returns it.
- parseReqTable: Parses a requirements table and reads each row into a Req structure, returned in a slice.
*/
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
	// The valid certification document filename format
	reCertdoc = regexp.MustCompile(`^(\w+)-(\d+)-(\w+)$`)

	// For detecting ATX Headings, see http://spec.commonmark.org/0.27/#atx-headings
	reATXHeading = regexp.MustCompile(`(?m)^ {0,3}(#{1,6})( +(.*)( #* *)?)?$`)

	// For detecting the first row and delimiter row of a requirement table
	reTableHeader    = regexp.MustCompile(`(?m)^\| *ID *\|(?:[^\|]*\|)+$`)
	reTableDelimiter = regexp.MustCompile(`(?m)^\|(?: *-+ *\|)+$`)

	// REQ, project number, project abbreviation, req type, req number
	// For example: REQ-TRAQ-SWH-4
	reReqIdStr   = `REQ-(\w+)-(\w+)-(\d+)`
	ReReqID      = regexp.MustCompile(reReqIdStr)
	reReqIDBad   = regexp.MustCompile(`(?i)REQ-((\d+)|((\w+)-(\d+)))`)
	ReReqDeleted = regexp.MustCompile(reReqIdStr + ` DELETED`)

	// For detecting attributes sections and attributes
	reAttributesSectionHeading = regexp.MustCompile(`(?m)\n#{2,6} Attributes:$`)
	reReqKWD                   = regexp.MustCompile(`(?i)- ([^:]+): `)
)

// ReqType defines what type of requirement we are parsing. None, a heading based requirement or a table of
// requirements.
type ReqType int

const (
	None ReqType = iota
	Heading
	Table
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

// parseMarkdown parses a certification document and returns the found requirements.
// @llr REQ-TRAQ-SWL-21, REQ-TRAQ-SWL-24
func parseMarkdown(f string) ([]*Req, error) {
	var (
		reqs []*Req

		lastHeadingLevel int // The level of the last ATX heading.
		lastHeadingLine  int // The line number of the last ATX heading.
		reqLevel         int // The level of the ATX heading starting the requirement.
		reqLine          int // The line number of the ATX heading starting the requirement.

		reqBuf bytes.Buffer // Temporary buffer for the fragment being read in.
		inReq  ReqType      // The type of fragment being read.
	)

	r, err := os.Open(f)
	if err != nil {
		return nil, err
	}
	scan := bufio.NewScanner(r)

	// scan through the markdown, one line at a time
	for lno := 1; scan.Scan(); lno++ {
		line := scan.Text()

		// check if we've hit an ATX heading or the first row of a requirements table
		if reATXHeading.MatchString(line) {
			// it's an ATX heading
			ATXparts := reATXHeading.FindStringSubmatch(line)
			level := len(ATXparts[1])
			title := ATXparts[3]
			reqIDs := ReReqID.FindAllString(title, -1)
			if len(reqIDs) > 1 {
				return nil, fmt.Errorf("malformed requirement title: too many IDs on line %d: %q", lno, line)
			}
			headingHasReqID := len(reqIDs) == 1

			// Check this heading is at the correct level given it's position in the document
			if inReq == Heading {
				// A request is currently being parsed.
				if headingHasReqID {
					// This is a requirement heading.
					// The level must be the same as the current requirement.
					if level != reqLevel {
						return nil, fmt.Errorf("requirement heading on line %d must be at same level as requirement heading on line %d (%d != %d): %q", lno, reqLine, level, reqLevel, line)
					}
				} else {
					// No requirement ID on this heading.
					// The heading level must be lower or higher than the current
					// requirement's heading level. We don't want to mix requirements
					// with other headings of the same level, in the same section.
					if level == reqLevel {
						return nil, fmt.Errorf("non-requirement heading on line %d at same level as requirement heading on line %d (%d): %q", lno, reqLine, level, line)
					}
				}
			} else {
				// Nothing going on yet.
				if headingHasReqID {
					// Can be the first one or the first one in another section.
					if level == lastHeadingLevel {
						return nil, fmt.Errorf("requirement heading on line %d at same level as previous heading on line %d (%d): %q", lno, lastHeadingLine, level, line)
					}
				}
			}

			// If we're currently parsing a requirement and it's appropriate close it
			if (inReq != None) && (headingHasReqID || level < reqLevel) {
				reqs, err = parseMarkdownFragment(inReq, reqBuf.String(), reqs)
				inReq = None
			}

			// If this is the start of a new requirement initialise data
			if headingHasReqID {
				inReq = Heading
				reqLevel = level
				reqLine = lno
				reqBuf.Reset()
				line = title
			}
			if level > 0 {
				lastHeadingLevel = level
				lastHeadingLine = lno
			}
		} else if reTableHeader.MatchString(line) {
			// It's a requirements table
			// If we're currently parsing a requirement close it
			if inReq != None {
				reqs, err = parseMarkdownFragment(inReq, reqBuf.String(), reqs)
			}
			// Start a new requirement table
			inReq = Table
			reqBuf.Reset()
		}

		if inReq != None {
			reqBuf.WriteString(line)
			reqBuf.WriteString("\n")
		}
	}
	if err := scan.Err(); err != nil {
		return nil, err
	}

	if inReq != None {
		// Close the current requirement, we're at the end.
		reqs, err = parseMarkdownFragment(inReq, reqBuf.String(), reqs)
	}

	return reqs, nil
}

// parseMarkdownFragment accepts a string containing either an ATX requirement or a requirements table and calls the
// appropriate parsing function
func parseMarkdownFragment(reqType ReqType, txt string, reqs []*Req) ([]*Req, error) {

	if reqType == Heading {
		// An ATX requirement
		newReq, err := parseReq(txt)
		if err != nil {
			return reqs, err
		}
		reqs = append(reqs, newReq)
	} else {
		var err error
		// A requirements table
		reqs, err = parseReqTable(txt, reqs)
		if err != nil {
			return reqs, err
		}
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

	ID, IDNumber, err := extractIDAndNum(txt)
	if err != nil {
		return nil, err
	}

	r := &Req{
		ID:         ID,
		IDNumber:   IDNumber,
		Attributes: map[string]string{},
	}

	var ok bool
	if r.Level, ok = config.ReqTypeToReqLevel[r.ReqType()]; !ok {
		return nil, fmt.Errorf("Invalid request type: %q", r.ReqType())
	}

	// chop defining ID and any punctuation
	txt = strings.TrimPrefix(txt, ID)
	txt = strings.TrimLeftFunc(txt, IsPunctOrSpace)

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
	err = parseParents(r)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// parseReqTable reads a table of requirements one row at a time and parses the content into Req structures which are
// then returned in a slice.
//
// Tables have the following format:
// | ID | Title | Body | Attribute1 | Attribute2 |
// | --- | --- | --- | --- | --- |
// | ReqID | <text> | <text> | <text> | <text> |
//
// The first column must be "ID" and each row must contain a valid ReqID. Other columns are optional.
//
// @llr REQ-TRAQ-SWL-25
func parseReqTable(txt string, reqs []*Req) ([]*Req, error) {

	var attributes []string

	// Split the table into rows and loop through
	for rowN, rowS := range strings.Split(txt, "\n") {

		// The first row contains the attribute names for each column, the first column must be "ID"
		if rowN == 0 {
			if reTableHeader.MatchString(rowS) {
				attributes = splitTableLine(rowS)
				for i, a := range attributes {
					attributes[i] = strings.ToUpper(a)
				}
			} else {
				return reqs, fmt.Errorf("first column must be \"ID\"")
			}
		} else {
			if reTableDelimiter.MatchString(rowS) {
				// Ignore the delimiter row
				continue
			}

			values := splitTableLine(rowS)

			if len(values) == 0 {
				// End of table
				break
			}

			if len(values) < len(attributes) {
				return reqs, fmt.Errorf("too few cells on row %d of requirement table", rowN+1)
			}

			r := &Req{Attributes: map[string]string{}}

			// For each attribute in the first row, read in the associated value on this row
			for i, k := range attributes {
				if k == "ID" {
					ID, IDNumber, err := extractIDAndNum(values[i])
					if err != nil {
						return reqs, err
					}
					r.ID = ID
					r.IDNumber = IDNumber
				} else if k == "TITLE" {
					r.Title = values[i]
				} else if k == "BODY" {
					r.Body = values[i]
				} else {
					if k == "PARENT" {
						// make our lives easier, accept both, output only PARENTS
						k = "PARENTS"
					}

					if values[i] != "" {
						r.Attributes[k] = values[i]
					}
				}
			}

			err := parseParents(r)
			if err != nil {
				return reqs, err
			}

			reqs = append(reqs, r)
		}
	}

	return reqs, nil
}

// splitTableLine splits a pipe table row in cells. Does not count for
// escaped `|` characters or other cases when `|` should not be considered
// a cell separator. Removes the first and last parts if they are empty.
func splitTableLine(line string) []string {
	// The `|` at the beginning of the line is ignored because it
	// represents visually the table's left side.
	parts := strings.Split(line, "|")
	if parts[0] == "" {
		parts = parts[1:]
	}
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	// Trim the space from each cell.
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	return parts
}

// extractIDAndNum parses a requirement identifier string and returns the ID string and sequence number
func extractIDAndNum(reqStr string) (string, int, error) {
	head := reqStr
	if len(head) > 40 {
		head = head[:40]
	}
	defid := ReReqID.FindStringSubmatchIndex(reqStr)
	if len(defid) == 0 {
		if reReqIDBad.MatchString(reqStr) {
			return "", 0, fmt.Errorf("malformed requirement: found only malformed ID: %q (doesn't match %q)", head, ReReqID)
		}
		return "", 0, fmt.Errorf("malformed requirement: missing ID in first 40 characters: %q", head)
	}

	if defid[0] > 0 {
		return "", 0, fmt.Errorf("malformed requirement: ID must be at the start of the title: %q", head)
	}

	IDNumber, err := strconv.Atoi(reqStr[defid[6]:defid[7]])
	if err != nil {
		return "", 0, err
	}
	return reqStr[defid[0]:defid[1]], IDNumber, nil
}

// parseParents validates the Parents attribute of a requirement
func parseParents(r *Req) error {
	// PARENTS must be punctuation/space separated list of parseable req-ids.
	parents := r.Attributes["PARENTS"]
	parmatch := ReReqID.FindAllStringSubmatchIndex(parents, -1)

	var parentIDs []string

	for i, ids := range parmatch {
		val := parents[ids[0]:ids[1]]
		parentIDs = append(parentIDs, val)
		if i > 0 {
			sep := parents[parmatch[i-1][1]:ids[0]]
			if strings.TrimFunc(sep, IsPunctOrSpace) != "" {
				return fmt.Errorf("requirement %s parents: unparseable as list of requirement ids: %q in %q", r.ID, sep, parents)
			}
		}
	}
	r.ParentIds = parentIDs
	return nil
}

func IsPunctOrSpace(r rune) bool {
	return unicode.IsSpace(r) || unicode.IsPunct(r)
}
