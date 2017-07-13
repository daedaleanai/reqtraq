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
	// For example: REQ-TRAQ-SWH-4
	reReqIdStr                 = `REQ-(\w+)-(SYS|SWH|SWL|HWH|HWL)-(\d+)`
	ReReqID                    = regexp.MustCompile(reReqIdStr)
	reReqIDBad                 = regexp.MustCompile(`(?i)REQ-((\d+)|((\w+)-(\d+)))`)
	ReReqDeleted               = regexp.MustCompile(reReqIdStr + ` DELETED`)
	reAttributesSectionHeading = regexp.MustCompile(`(?m)\n#{2,6} Attributes:$`)
	reReqKWD                   = regexp.MustCompile(`(?i)- ([^:]+): `)
)

// formatBodyAsHTML converts a string containing markdown to HTML using pandoc.
// @llr REQ-TRAQ-SWL-19
func formatBodyAsHTML(txt string) template.HTML {
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
//
// @llr REQ-TRAQ-SWL-13
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

	if defid[0] > 0 {
		return nil, fmt.Errorf("malformed requirement: ID must be at the start of the title: %q", head)
	}

	r := &Req{
		ID:         txt[defid[0]:defid[1]],
		Attributes: map[string]string{},
	}

	// chop defining ID and any punctuation
	txt = strings.TrimLeftFunc(txt[defid[1]:], IsPunctOrSpace)

	// The first line is the title.
	parts := strings.SplitN(strings.TrimSpace(txt), "\n", 2)
	if len(parts) < 2 {
		// It means there is a single line.
		return nil, fmt.Errorf("Requirement must not be empty: %s", r.ID)
	}
	r.Title = parts[0]

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

	r.Body = formatBodyAsHTML(bodyAndAttributes[:attributesStart])
	if strings.TrimSpace(string(r.Body)) == "" {
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

	level, ok := config.ReqTypeToReqLevel[r.ReqType()]
	if !ok {
		return nil, fmt.Errorf("Invalid request type: %q", r.ReqType())
	}
	r.Level = level

	return r, nil
}

func IsPunctOrSpace(r rune) bool {
	return unicode.IsSpace(r) || unicode.IsPunct(r)
}
