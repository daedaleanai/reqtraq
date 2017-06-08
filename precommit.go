// @llr REQ-0-DDLN-SWL-003
// @llr REQ-0-DDLN-SWL-005
package main

import (
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/lyx"
)

// lintLyxReq is called for each requirement while building the req graph
func lintLyxReq(fileName string, nReqs int, isReqPresent []bool, r *lyx.Req) []error {

	// extract file name without extension
	fNameWithExt := path.Base(fileName)
	extension := filepath.Ext(fNameWithExt)
	fName := fNameWithExt[0 : len(fNameWithExt)-len(extension)]

	// figure out req type from doc type
	fNameComps := strings.Split(fName, "-")
	docType := fNameComps[len(fNameComps)-1]
	reqType := config.DocTypeToReqType[docType]

	var errs []error
	reqIdComps := strings.Split(r.ID, "-")
	// check requirement name
	if reqIdComps[0] != "REQ" {
		errs = append(errs, fmt.Errorf("Incorrect requirement name %s. Every requirement needs to start with REQ, got %s.", r.ID, reqIdComps[0]))
	}
	if reqIdComps[1] != fNameComps[0] {
		errs = append(errs, fmt.Errorf("Incorrect project ID for requirement %s. Expected %s, got %s.", r.ID, fNameComps[0], reqIdComps[1]))
	}
	if reqIdComps[2] != fNameComps[1] {
		errs = append(errs, fmt.Errorf("Incorrect project abbreviation for requirement %s. Expected %s, got %s.", r.ID, fNameComps[1], reqIdComps[2]))
	}
	if reqIdComps[3] != reqType {
		errs = append(errs, fmt.Errorf("Incorrect requirement type for requirement %s. Expected %s, got %s.", r.ID, reqType, reqIdComps[3]))
	}
	currentId, err2 := strconv.Atoi(reqIdComps[len(reqIdComps)-1])
	if err2 != nil {
		errs = append(errs, fmt.Errorf("Invalid requirement sequence number for %s (failed to parse): %s", r.ID, reqIdComps[len(reqIdComps)-1]))
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
