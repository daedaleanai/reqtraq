/*
   Functions related to the handling of requirements and code tags.

   The following types and associated methods are implemented:
     ReqGraph - The complete information about a set of requirements and associated code tags.
     Req - A requirement node in the graph of requirements.
     Schema - The information held in the schema file defining the rules that the requirement graph must follow.
     byPosition, byIDNumber and ByFilenameTag - Provides sort functions to order requirements or code,
     ReqFilter - The different parameters used to filter the requirements set.
*/

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/git"
	"github.com/daedaleanai/reqtraq/linepipes"
	"github.com/pkg/errors"
)

// ReqGraph holds the complete information about a set of requirements and associated code tags.
type ReqGraph struct {
	// Reqs contains the requirements by ID.
	Reqs map[string]*Req
	// CodeTags contains the source code functions per file.
	// The keys are paths relative to the git repo path.
	CodeTags map[string][]*Code
	// Errors which have been found while analyzing the graph.
	// This is extended in multiple places.
	Errors []error
	// Schema holds information about what a valid ReqGraph looks like e.g. valid attributes
	Schema Schema
}

// CreateReqGraph returns a graph resulting from parsing the certdocs. The graph includes a list of
// errors found while walking the requirements, code, or resolving the graph.
// The separate returned error indicates if reading the certdocs and code failed.
// @llr REQ-TRAQ-SWL-1
func CreateReqGraph(certdocsPath, codePath, schemaPath string) (*ReqGraph, error) {
	rg := &ReqGraph{make(map[string]*Req, 0), nil, make([]error, 0), Schema{}}

	// Walk through the documents.
	err := filepath.Walk(filepath.Join(git.RepoPath(), certdocsPath),
		func(fileName string, info os.FileInfo, err error) error {
			if strings.ToLower(path.Ext(fileName)) == ".md" {
				err = rg.addCertdocToGraph(fileName)
				if err != nil {
					return err
				}
			}
			return nil
		})
	if err != nil {
		return rg, errors.Wrap(err, "failed walking certdocs")
	}

	// Find and parse the code files.
	rg.CodeTags, err = ParseCode(codePath)
	if err != nil {
		return rg, err
	}

	// Load the schema, so we can use it to validate attributes
	rg.Schema, err = ParseSchema(schemaPath)
	if err != nil {
		return rg, err
	}

	// Call resolve to check links between requirements and code
	rg.Errors = append(rg.Errors, rg.resolve()...)

	return rg, nil
}

// AddReq appends the requirements list with a new requirement, after confirming that it's not already present
// @llr REQ-TRAQ-SWL-28
func (rg *ReqGraph) AddReq(req *Req, path string) error {
	if v := rg.Reqs[req.ID]; v != nil {
		return fmt.Errorf("Requirement %s in %s already defined in %s", req.ID, path, v.Path)
	}
	req.Path = strings.TrimPrefix(path, git.RepoPath())

	rg.Reqs[req.ID] = req
	return nil
}

// addCertdocToGraph parses a file for requirements, checks their validity and then adds them along with any errors
// found to the regGraph
// @llr REQ-TRAQ-SWL-27
func (rg *ReqGraph) addCertdocToGraph(fileName string) error {
	reqs, err := ParseCertdoc(fileName)
	if err != nil {
		return fmt.Errorf("error parsing %s: %v", fileName, err)
	}
	if len(reqs) == 0 {
		return nil
	}

	// sort the requirements so we can check the sequence
	sort.Sort(byIDNumber(reqs))

	isReqPresent := make([]bool, reqs[len(reqs)-1].IDNumber)
	isAsmPresent := make([]bool, reqs[len(reqs)-1].IDNumber)
	nextReqId := 1
	nextAsmId := 1

	for i, r := range reqs {
		var newErrs []error
		if r.Prefix == "REQ" {
			newErrs = r.checkID(fileName, nextReqId, isReqPresent)
			nextReqId = r.IDNumber + 1
		} else if r.Prefix == "ASM" {
			newErrs = r.checkID(fileName, nextAsmId, isAsmPresent)
			nextAsmId = r.IDNumber + 1
		}
		if len(newErrs) != 0 {
			rg.Errors = append(rg.Errors, newErrs...)
			continue
		}
		r.Position = i
		rg.AddReq(r, fileName)
	}
	return nil
}

// resolve walks the requirements graph and resolves the links between different levels of requirements
// and with code tags. References to requirements within requirements text is checked as well as validity
// of attributes against the schema. Any errors encountered such as links to non-existent requirements
// are returned.
// @llr REQ-TRAQ-SWL-10, REQ-TRAQ-SWL-11
func (rg *ReqGraph) resolve() []error {
	errs := make([]error, 0)

	// Walk the requirements, resolving links and looking for errors
	for _, req := range rg.Reqs {
		if req.IsDeleted() {
			continue
		}

		// Validate attributes
		errs = append(errs, req.checkAttributes(rg.Schema)...)

		// Validate parent links of requirements
		for _, parentID := range req.ParentIds {
			parent := rg.Reqs[parentID]
			if parent != nil {
				if parent.IsDeleted() {
					errs = append(errs, errors.New("Invalid parent of requirement "+req.ID+": "+parentID+" is deleted."))
				}
				parent.Children = append(parent.Children, req)
				req.Parents = append(req.Parents, parent)
			} else {
				errs = append(errs, errors.New("Invalid parent of requirement "+req.ID+": "+parentID+" does not exist."))
			}
		}
		// Validate references to requirements in body text
		matches := ReReqID.FindAllStringSubmatchIndex(req.Body, -1)
		for _, ids := range matches {
			reqID := req.Body[ids[0]:ids[1]]
			v, reqFound := rg.Reqs[reqID]
			if !reqFound {
				errs = append(errs, fmt.Errorf("Invalid reference to non existent requirement %s in body of %s.", reqID, req.ID))
			} else if v.IsDeleted() {
				errs = append(errs, fmt.Errorf("Invalid reference to deleted requirement %s in body of %s.", reqID, req.ID))
			}
		}

		// TODO check for references to missing or deleted requirements in attribute text
	}

	// Walk the code tags, resolving links and looking for errors
	for _, tags := range rg.CodeTags {
		for _, code := range tags {
			if len(code.ParentIds) == 0 {
				errs = append(errs, fmt.Errorf("Function %s@%s:%d has no parents.", code.Tag, code.Path, code.Line))
			}
			for _, parentID := range code.ParentIds {
				parent := rg.Reqs[parentID]
				if parent != nil {
					if parent.IsDeleted() {
						errs = append(errs, fmt.Errorf("Invalid reference in function %s@%s:%d, %s is deleted.", code.Tag, code.Path, code.Line, parentID))
					}
					if parent.Level == config.LOW {
						parent.Tags = append(parent.Tags, code)
						code.Parents = append(code.Parents, parent)
					} else {
						errs = append(errs, fmt.Errorf("Invalid reference in function %s@%s:%d, %s is not a low-level requirement.", code.Tag, code.Path, code.Line, parentID))
					}
				} else {
					errs = append(errs, fmt.Errorf("Invalid reference in function %s@%s:%d, %s does not exist.", code.Tag, code.Path, code.Line, parentID))
				}
			}
		}
	}

	if len(errs) > 0 {
		return errs
	}

	for _, req := range rg.Reqs {
		sort.Sort(byPosition(req.Parents))
		sort.Sort(byPosition(req.Children))
	}

	return nil
}

// Req represents a requirement node in the graph of requirements.
type Req struct {
	ID       string                  // e.g. REQ-TEST-SWL-1
	Prefix   string                  // e.g. REQ
	IDNumber int                     // e.g. 1
	Level    config.RequirementLevel // e.g. LOW
	// Path identifies the file this was found in relative to repo root.
	Path      string
	ParentIds []string
	// Parents holds the parent requirements.
	Parents []*Req
	// Children holds the children requirements.
	Children []*Req
	// Tags holds the associated code functions.
	Tags  []*Code
	Title string
	Body  string
	// Attributes of the requirement by uppercase name.
	Attributes map[string]string
	Position   int
}

// Changelists generates a list of Phabicator revisions that have affected a requirement
// TODO: except it doesn't really, it actually returns a list of revisions which have
//   affected the file(s) that this requirement is referenced in, whether the associated
//   function is affected or not. which is basically useless.
// @llr REQ-TRAQ-SWL-22
func (r *Req) Changelists() map[string]string {
	var reDiffRev = regexp.MustCompile(`Differential Revision:\s(.*)\s`)

	m := map[string]string{}
	if r.Level == config.LOW {
		for _, c := range r.Tags {
			// TODO: this should go through the git package
			res, err := linepipes.All(linepipes.Run("git", "log", c.Path))
			if err != nil {
				log.Fatal(err)
			}

			matches := reDiffRev.FindAllStringSubmatch(res, -1)
			if len(matches) < 1 {
				log.Printf("Could not extract differential revision for file: %s", c.Path)
				log.Println("Newly added?")
			}

			for _, match := range matches {
				if len(match) != 2 {
					log.Fatal("Count not extract changelist substring for filepath: ", c.Path)
				}
				fields := strings.Split(match[1], "/")
				m[fields[len(fields)-1]] = match[1]
			}
		}
	}
	return m
}

// IsDeleted checks if the requirement title starts with 'DELETED'
// @llr REQ-TRAQ-SWL-23
func (r *Req) IsDeleted() bool {
	return strings.HasPrefix(r.Title, "DELETED")
}

// checkAttributes validates the requirement attributes against the provided schema, returns a list of errors found
// @llr REQ-TRAQ-SWL-10, REQ-TRAQ-SWL-29
func (r *Req) checkAttributes(as Schema) []error {
	var errs []error
	var anyAttributes []string
	anyCount := 0

	// Iterate the attribute rules
	for name, rules := range as.Attributes {
		for _, rule := range rules {

			if !rule.Filter.MatchString(r.ID) {
				continue
			}

			if rule.Any {
				anyAttributes = append(anyAttributes, name)
			}

			reqValue, reqValuePresent := r.Attributes[strings.ToUpper(name)]
			reqValuePresent = reqValuePresent && reqValue != ""

			if !reqValuePresent && rule.Required {
				errs = append(errs, fmt.Errorf("Requirement '%s' is missing attribute '%s'.", r.ID, name))
			} else if reqValuePresent {
				if rule.Any {
					anyCount++
				}

				if !rule.Value.MatchString(reqValue) {
					errs = append(errs, fmt.Errorf("Requirement '%s' has invalid value '%s' in attribute '%s'.", r.ID, reqValue, name))
				}
			}
		}
	}

	if len(anyAttributes) > 0 && anyCount == 0 {
		sort.Strings(anyAttributes)
		errs = append(errs, fmt.Errorf("Requirement '%s' is missing at least one of the attributes '%s'.", r.ID, strings.Join(anyAttributes, ",")))
	}

	// Iterate the requirement attributes to check for unknown ones
	for name := range r.Attributes {
		if _, present := as.Attributes[strings.ToUpper(name)]; !present {
			errs = append(errs, fmt.Errorf("Requirement '%s' has unknown attribute '%s'.", r.ID, name))
		}
	}

	return errs
}

// checkID validates each part of the requirement ID
// @llr REQ-TRAQ-SWL-25, REQ-TRAQ-SWL-26
func (r *Req) checkID(fileName string, expectedIDNumber int, isReqPresent []bool) []error {
	// extract file name without extension
	fNameWithExt := path.Base(fileName)
	extension := filepath.Ext(fNameWithExt)
	fName := fNameWithExt[0 : len(fNameWithExt)-len(extension)]

	// figure out req type from doc type
	fNameComps := strings.Split(fName, "-")
	docType := fNameComps[len(fNameComps)-1]
	reqType := config.DocTypeToReqType[docType]

	var errs []error
	reqIDComps := strings.Split(r.ID, "-") // results in an array such as [REQ PROJECT REQTYPE 1234]
	// check requirement name, no need to check prefix because it would not have been parsed otherwise
	if reqIDComps[1] != fNameComps[0] {
		errs = append(errs, fmt.Errorf("Incorrect project abbreviation for requirement %s. Expected %s, got %s.", r.ID, fNameComps[0], reqIDComps[1]))
	}
	if reqIDComps[2] != reqType {
		errs = append(errs, fmt.Errorf("Incorrect requirement type for requirement %s. Expected %s, got %s.", r.ID, reqType, reqIDComps[2]))
	}
	if reqIDComps[3][0] == '0' {
		errs = append(errs, fmt.Errorf("Requirement number cannot begin with a 0: %s. Got %s.", r.ID, reqIDComps[3]))
	}

	currentID, err2 := strconv.Atoi(reqIDComps[3])
	if err2 != nil {
		errs = append(errs, fmt.Errorf("Invalid requirement sequence number for %s (failed to parse): %s", r.ID, reqIDComps[3]))
	} else {
		if currentID < 1 {
			errs = append(errs, fmt.Errorf("Invalid requirement sequence number for %s: first requirement has to start with 001.", r.ID))
		} else {
			if isReqPresent[currentID-1] {
				errs = append(errs, fmt.Errorf("Invalid requirement sequence number for %s, is duplicate.", r.ID))
			} else {
				if currentID != expectedIDNumber {
					errs = append(errs, fmt.Errorf("Invalid requirement sequence number for %s: missing requirements in between. Expected ID Number %d.", r.ID, expectedIDNumber))
				}
			}
			isReqPresent[currentID-1] = true
		}
	}

	return errs
}

type AttributeRule struct {
	Filter   *regexp.Regexp // holds a regex which matches which requirement IDs this rule applies to
	Required bool           // indicates if this attribute is mandatory
	Any      bool           // indicates if this attribute, or another with this flag set, is mandatory
	Value    *regexp.Regexp // regex which matches valid values for the attribute
}

// Schema holds the information held in the schema file defining the rules that the requirement graph must follow.
type Schema struct {
	Attributes map[string]map[string]AttributeRule
}

// ParseSchema loads and returns the requirements schema from the specified file
// @llr REQ-TRAQ-SWL-29
func ParseSchema(schemaPath string) (Schema, error) {
	var schema Schema
	schemaJson := struct {
		Attributes []map[string]string
	}{}

	b, err := ioutil.ReadFile(schemaPath)
	if err != nil {
		return schema, fmt.Errorf("Schema file missing: %s", schemaPath)
	}
	if err := json.Unmarshal(b, &schemaJson); err != nil {
		return schema, fmt.Errorf("Error while parsing schema: %s", err)
	}

	schema.Attributes = make(map[string]map[string]AttributeRule)

	for _, a := range schemaJson.Attributes {
		name, namePresent := a["name"]
		filter, filterPresent := a["filter"]
		required, requiredPresent := a["required"]
		value, valuePresent := a["value"]

		if !namePresent {
			return schema, fmt.Errorf("Schema %s contains attributes rule with no name", schemaPath)
		}
		name = strings.ToUpper(name)
		if _, present := schema.Attributes[name]; !present {
			schema.Attributes[name] = make(map[string]AttributeRule)
		}

		var newRule AttributeRule

		if !filterPresent {
			filter = ".*"
		}

		if _, present := schema.Attributes[name][filter]; present {
			return schema, fmt.Errorf("Schema %s contains duplicate attribute name/filter: %s/%s", schemaPath, name, filter)
		}

		newRule.Filter, err = regexp.Compile(filter)
		if err != nil {
			return schema, fmt.Errorf("Schema %s contains invalid regex: %s", schemaPath, filter)
		}

		newRule.Required = true
		newRule.Any = false
		if requiredPresent {
			switch required {
			case "false":
				newRule.Required = false
				newRule.Any = false
			case "any":
				newRule.Required = false
				newRule.Any = true
			case "true":
				// default
			default:
				return schema, fmt.Errorf("Schema %s contains invalid require flag: %s", schemaPath, required)
			}
		}

		if valuePresent {
			newRule.Value, err = regexp.Compile(value)
			if err != nil {
				return schema, fmt.Errorf("Schema %s contains invalid regex: %s", schemaPath, value)
			}
		} else {
			newRule.Value, _ = regexp.Compile(".*")
		}

		schema.Attributes[name][filter] = newRule
	}

	return schema, nil
}

// byPosition provides sort functions to order requirements by their Position value
type byPosition []*Req

// @llr REQ-TRAQ-SWL-45
func (a byPosition) Len() int { return len(a) }

// @llr REQ-TRAQ-SWL-45
func (a byPosition) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// @llr REQ-TRAQ-SWL-45
func (a byPosition) Less(i, j int) bool { return a[i].Position < a[j].Position }

// byIDNumber provides sort functions to order requirements by their IDNumber value
type byIDNumber []*Req

// @llr REQ-TRAQ-SWL-46
func (a byIDNumber) Len() int { return len(a) }

// @llr REQ-TRAQ-SWL-46
func (a byIDNumber) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// @llr REQ-TRAQ-SWL-46
func (a byIDNumber) Less(i, j int) bool { return a[i].IDNumber < a[j].IDNumber }

// byFilenameTag provides sort functions to order code by their path value, and then line number
type byFilenameTag []*Code

// @llr REQ-TRAQ-SWL-47
func (a byFilenameTag) Len() int { return len(a) }

// @llr REQ-TRAQ-SWL-47
func (a byFilenameTag) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// @llr REQ-TRAQ-SWL-47
func (a byFilenameTag) Less(i, j int) bool {
	switch strings.Compare(a[i].Path, a[j].Path) {
	case -1:
		return true
	case 0:
		return a[i].Line < a[j].Line
	}
	return false
}

// ReqFilter holds the different parameters used to filter the requirements set.
type ReqFilter struct {
	IDRegexp           *regexp.Regexp
	TitleRegexp        *regexp.Regexp
	BodyRegexp         *regexp.Regexp
	AnyAttributeRegexp *regexp.Regexp
	AttributeRegexp    map[string]*regexp.Regexp
}

// IsEmpty returns whether the filter has no restriction.
// @llr REQ-TRAQ-SWL-20, REQ-TRAQ-SWL-21
func (f ReqFilter) IsEmpty() bool {
	return f.IDRegexp == nil && f.TitleRegexp == nil &&
		f.BodyRegexp == nil && f.AnyAttributeRegexp == nil &&
		len(f.AttributeRegexp) == 0
}
