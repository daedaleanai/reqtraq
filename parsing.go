// @llr REQ-0-DDLN-SWL-001
package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"os/exec"
	"regexp"
	"strings"
	"unicode"

	"github.com/daedaleanai/reqtraq/config"
)

var (
	// REQ, project number, project abbreviation, req type, req number
	// For example: REQ-0-DDLN-SWH-004
	reReqIdStr   = `REQ-(\d+)-(\w+)-(SYS|SWH|SWL|HWH|HWL)-(\d+)`
	ReReqID      = regexp.MustCompile(reReqIdStr)
	ReReqDeleted = regexp.MustCompile(reReqIdStr + ` DELETED`)
	reReqIDBad   = regexp.MustCompile(`(?i)REQ(-(\w+))+`)
	reReqKWD     = regexp.MustCompile(`(?i)(- )?(rationale|parent|parents|safety impact|verification|urgent|important|mode|provenance):`)
)

// Given a string containing markdown, convert it to HTML using pandoc
func formatBodyAsHTML(txt string) (template.HTML) {
	cmd := exec.Command("pandoc", "--mathjax")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal("Couldn't get input pipe for pandoc: ", err)
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, txt)
	}()

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal("Error while running pandoc: ", err)
	}

	return template.HTML(out)
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
	lyx := strings.HasPrefix(txt, "\n")
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

	if lyx {
		if defid[0] > 20 {
			return nil, fmt.Errorf("malformed requirement: too much heading garbage before ID: %q", head)
		}
	} else {
		if defid[0] > 0 {
			return nil, fmt.Errorf("malformed requirement: ID must be at the start of the title: %q", head)
		}
	}

	r := &Req{
		ID:         txt[defid[0]:defid[1]],
		Attributes: map[string]string{},
	}

	// chop defining ID and any punctuation
	txt = strings.TrimLeftFunc(txt[defid[1]:], unicode.IsPunct)
	txt = strings.TrimLeftFunc(txt, unicode.IsSpace)

	var attributesStart int
	kwdMatches := reReqKWD.FindAllStringSubmatchIndex(txt, -1)
	if len(kwdMatches) == 0 {
		return nil, fmt.Errorf("requirement %s contains no attributes", r.ID)
	}
	if lyx {
		attributesStart = kwdMatches[0][0]
	} else {
		attributesStart = strings.Index(txt, "\n###### Attributes:\n")
	}
	for i, v := range kwdMatches {
		key := strings.ToUpper(txt[v[4]:v[5]])
		if key == "PARENT" { // make our lives easier, accept both, output only PARENTS
			key = "PARENTS"
		}
		e := len(txt)
		if i < len(kwdMatches)-1 {
			e = kwdMatches[i+1][0]
		}
		if _, ok := r.Attributes[key]; ok {
			return nil, fmt.Errorf("requirement %s contains duplicate attribute: %q", r.ID, key)
		}
		r.Attributes[key] = strings.TrimSpace(txt[v[1]:e])
	}

	// TEXT is anything up to the first keyword we found
	txt = txt[:attributesStart]

	// PARENTS must be punctuation/space separated list of parseable req-ids.
	parents := r.Attributes["PARENTS"]
	parmatch := ReReqID.FindAllStringSubmatchIndex(parents, -1)
	for i, ids := range parmatch {
		val := parents[ids[0]:ids[1]]
		r.ParentIds = append(r.ParentIds, val)
		if i > 0 {
			sep := parents[parmatch[i-1][1]:ids[0]]
			if strings.TrimFunc(sep, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsPunct(r) }) != "" {
				return nil, fmt.Errorf("requirement %s parents: unparseable as list of requirement ids: %q in %q", r.ID, sep, parents)
			}
		}
	}

	level, ok := config.ReqTypeToReqLevel[r.ReqType()]
	if !ok {
		return nil, fmt.Errorf("Invalid request type: %q", r.ReqType())
	}
	r.Level = level

	parts := strings.SplitN(strings.TrimSpace(txt), "\n", 2)
	r.Title = parts[0]

	r.Body = formatBodyAsHTML(parts[1])
	return r, nil
}
