// @llr REQ-TRAQ-SWL-3
// @llr REQ-TRAQ-SWL-5
package main

import (
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/daedaleanai/reqtraq/config"
)

// lintReq is called for each requirement while building the req graph
func lintReq(fileName string, nReqs int, isReqPresent []bool, r *Req) []error {
	// extract file name without extension
	fNameWithExt := path.Base(fileName)
	extension := filepath.Ext(fNameWithExt)
	fName := fNameWithExt[0 : len(fNameWithExt)-len(extension)]

	// figure out req type from doc type
	fNameComps := strings.Split(fName, "-")
	docType := fNameComps[len(fNameComps)-1]
	reqType := config.DocTypeToReqType[docType]

	var errs []error
	reqIdComps := strings.Split(r.ID, "-") // results in an array such as [REQ PROJECT REQTYPE 1234]
	// check requirement name
	if reqIdComps[0] != "REQ" {
		errs = append(errs, fmt.Errorf("Incorrect requirement name %s. Every requirement needs to start with REQ, got %s.", r.ID, reqIdComps[0]))
	}
	if reqIdComps[1] != fNameComps[0] {
		errs = append(errs, fmt.Errorf("Incorrect project abbreviation for requirement %s. Expected %s, got %s.", r.ID, fNameComps[0], reqIdComps[1]))
	}
	if reqIdComps[2] != reqType {
		errs = append(errs, fmt.Errorf("Incorrect requirement type for requirement %s. Expected %s, got %s.", r.ID, reqType, reqIdComps[2]))
	}
	if reqIdComps[3][0] == '0' {
		errs = append(errs, fmt.Errorf("Requirement number cannot begin with a 0: %s. Got %s.", r.ID, reqIdComps[3]))
	}

	currentId, err2 := strconv.Atoi(reqIdComps[3])
	if err2 != nil {
		errs = append(errs, fmt.Errorf("Invalid requirement sequence number for %s (failed to parse): %s", r.ID, reqIdComps[3]))
	} else {

		// check requirement sequence number
		if currentId > nReqs {
			errs = append(errs, fmt.Errorf("Invalid requirement sequence number for %s: missing requirements in between. Total number of requirements is %d.", r.ID, nReqs))
		} else {
			if currentId < 1 {
				errs = append(errs, fmt.Errorf("Invalid requirement sequence number for %s: first requirement has to start with 001.", r.ID))
			} else {
				if isReqPresent[currentId-1] {
					errs = append(errs, fmt.Errorf("Invalid requirement sequence number for %s, is duplicate.", r.ID))
				}
				isReqPresent[currentId-1] = true
			}
		}
	}

	return errs
}
