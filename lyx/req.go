// @llr REQ-0-DDLN-SWL-001
package lyx

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// Req represents a requirement.
type Req struct {
	ID         string
	Parents    []string
	ProjectID  string
	Attributes map[string]string
	Position   uint32
}

var (
	// REQ, project number, project abbreviation, req type, req number
	// For example: REQ-0-DDLN-SWH-004
	reReqIdStr   = `REQ-(\d+)-(\w+)-(SYS|SWH|SWL|HWH|HWL)-(\d+)`
	ReReqID      = regexp.MustCompile(reReqIdStr)
	ReReqDeleted = regexp.MustCompile(reReqIdStr + ` DELETED`)
	reReqIDBad   = regexp.MustCompile(`(?i)REQ(-(\w+))+`)
	reReqKWD     = regexp.MustCompile(`(?i)(rationale|parent|parents|safety\s+impact|verification|urgent|important|mode|provenance):`)
)

// Returns the requirement type for the given requirement, which is one of SYS, SWH, SWL, HWH, HWL or the empty string if
// the request is not initialized.
func (r *Req) ReqType() string {
	parts := ReReqID.FindStringSubmatch(r.ID)
	if len(parts) == 0 {
		return ""
	}
	return parts[3]
}

// ParseReq finds the first REQ-XXX tag and the reserved words and distills a Req from it.
//
// ParseReq parses according to the 'soft' format defined in the SRS:
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
// ParseReq does NOT validate the values or check if the mandatory attributes are set; use
// the Req.Check() method for that.
//
// Since the parsing is rather 'soft', ParseReq returns verbose errors indicating problems in
// a helpful way, meaning they at least provide enough context for the user to find the text.
func ParseReq(txt string) (*Req, error) {
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
	if defid[0] > 20 {
		return nil, fmt.Errorf("malformed requirement: too much heading garbage before ID: %q", head)
	}
	r := &Req{
		ID:         txt[defid[0]:defid[1]],
		ProjectID:  fmt.Sprintf("%s-%s", txt[defid[2]:defid[3]], strings.ToUpper(txt[defid[4]:defid[5]])),
		Attributes: map[string]string{},
	}

	// chop defining ID and any punctuation
	txt = strings.TrimLeftFunc(txt[defid[1]:], unicode.IsPunct)
	kwdMatches := reReqKWD.FindAllStringSubmatchIndex(txt, -1)
	// TEXT is anything up to the first keyword we found, or the entire remainder if we didn't.
	if len(kwdMatches) == 0 {
		r.Attributes["TEXT"] = txt
	} else {
		r.Attributes["TEXT"] = txt[:kwdMatches[0][0]]
	}
	for i, v := range kwdMatches {
		key := strings.ToUpper(txt[v[2]:v[3]])
		if key == "PARENT" { // make our lives easier, accept both, output only PARENTS
			key = "PARENTS"
		}
		e := len(txt)
		if i < len(kwdMatches)-1 {
			e = kwdMatches[i+1][0]
		}
		if _, ok := r.Attributes[key]; ok {
			return nil, fmt.Errorf("requirement %s contains repeated keyword: %q", r.ID, key)
		}
		r.Attributes[key] = strings.TrimSpace(txt[v[1]:e])
	}

	// PARENTS must be punctuation/space separated list of parseable req-ids.
	parents := r.Attributes["PARENTS"]
	parmatch := ReReqID.FindAllStringSubmatchIndex(parents, -1)
	for i, ids := range parmatch {
		val := parents[ids[0]:ids[1]]
		r.Parents = append(r.Parents, val)
		if i > 0 {
			sep := parents[parmatch[i-1][1]:ids[0]]
			if strings.TrimFunc(sep, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsPunct(r) }) != "" {
				return nil, fmt.Errorf("requirement %s parents: unparseable as list of requirement ids: %q in %q", r.ID, sep, parents)
			}
		}
	}

	return r, nil
}
