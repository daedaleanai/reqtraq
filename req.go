// @llr REQ-TRAQ-SWL-15
// @llr REQ-TRAQ-SWL-6
// @llr REQ-TRAQ-SWL-7
// @llr REQ-TRAQ-SWL-11
// @llr REQ-TRAQ-SWL-13

package main

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"html/template"
	"io"
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
	"github.com/daedaleanai/reqtraq/taskmgr"
)

type RequirementStatus int

const (
	NOT_STARTED RequirementStatus = iota // req does not have any children, unless code level
	STARTED                              // req does have children but incomplete
	COMPLETED                            // graph complete
)

var reqStatusToString = map[RequirementStatus]string{
	NOT_STARTED: "NOT STARTED",
	STARTED:     "STARTED",
	COMPLETED:   "COMPLETED",
}

func (rs RequirementStatus) String() string { return reqStatusToString[rs] }

var (
	// project abbreviation, certdoc type number, certdoc type
	reCertdoc = regexp.MustCompile(`^(\w+)-(\d+)-(\w+)$`)
	reDiffRev = regexp.MustCompile(`Differential Revision:\s(.*)\s`)
)

// Req represenents a Requirement Node in the graph of Requirements.
// The Attributes map has potential elements;
//  rationale safety_impact verification urgent important mode provenance
type Req struct {
	ID        string // code files do not have an ID, use Path as primary key
	Level     config.RequirementLevel
	Path      string // certification document or code file this was found in relative to repo root
	FileHash  string // for code files, the sha1 of the contents
	ParentIds []string
	Parents   []*Req
	Children  []*Req
	Title     string
	// Body contains various HTML tags (links, converted markdown, etc). Type must be HTML,
	// not a string, so it's not HTML-escaped by the templating engine.
	Body       template.HTML
	Attributes map[string]string
	Position   int
	Seen       bool
	Status     RequirementStatus
}

// Returns the requirement type for the given requirement, which is one of SYS, SWH, SWL, HWH, HWL or the empty string if
// the request is not initialized.
func (r *Req) ReqType() string {
	parts := ReReqID.FindStringSubmatch(r.ID)
	if len(parts) == 0 {
		return ""
	}
	return parts[2]
}

func (r *Req) resolveUp() {
	r.Seen = true
	for _, p := range r.Parents {
		p.resolveUp()
	}
}

func (r *Req) resolveDown() RequirementStatus {
	r.Seen = true
	r.Status = COMPLETED
	if r.Level != config.CODE && len(r.Children) == 0 {
		r.Status = NOT_STARTED
	} else {
		for _, c := range r.Children {
			if c.resolveDown() != COMPLETED {
				r.Status = STARTED
			}
		}
	}
	return r.Status
}

// IsDeleted checks if the requirement title starts with 'DELETED'
// @REQ-TRAQ-SWL-17
func (r *Req) IsDeleted() bool {
	return strings.HasPrefix(r.Title, "DELETED")
}

func (r *Req) CheckAttributes(as []map[string]string) []error {
	var errs []error
	if r.IsDeleted() {
		return errs
	}
	for _, a := range as {
		for k, v := range a {
			switch k {
			case "name":
				if _, ok := r.Attributes[strings.ToUpper(v)]; !ok {
					if !(r.Level == config.SYSTEM && strings.ToUpper(v) == "PARENTS") {
						errs = append(errs, fmt.Errorf("Requirement '%s' is missing attribute '%s'.", r.ID, v))
					}
				}
			case "value":
				aName := strings.ToUpper(a["name"])
				if _, ok := r.Attributes[aName]; ok {
					// attribute exists so needs to be valid
					expr, err := regexp.Compile(v) // TODO(dh) move out so only computed once for each value
					if err != nil {
						log.Fatal(err)
					}
					if !expr.MatchString(r.Attributes[aName]) {
						errs = append(errs, fmt.Errorf("Requirement '%s' has invalid value '%s' in attribute '%s'. Expected %s.", r.ID, r.Attributes[aName], aName, v))
					}
				}
			}
		}
	}
	return errs
}

func (r *Req) Tasklists() map[string]*taskmgr.Task {
	m := map[string]*taskmgr.Task{}
	projectID, err1 := taskmgr.TaskMgr.GetProject(config.ProjectName)
	if err1 != nil {
		log.Printf("Failed to get project '%s' from the task manager: %v", config.ProjectName, err1)
		return m
	}
	// Find and add primary task corresponding to Req
	task, err2 := taskmgr.TaskMgr.FindTask(r.ID, r.Title, projectID)
	if err2 != nil {
		log.Printf("Failed to find the task for requirement %s '%s' in project %s: %v", r.ID, r.Title, projectID, err2)
		return m
	}
	if task == nil {
		log.Printf("No task found for requirement %s '%s' in project %s", r.ID, r.Title, projectID)
		return m
	}
	m[task.ID] = task
	// Get all tasks that "task" depends on and add them
	for _, phid := range task.DependsOnTaskIDs {
		subTask, e := taskmgr.TaskMgr.FindTaskByID(phid)
		if e != nil {
			log.Printf("Failed to find subtask %s: %v", phid, e)
			continue
		}
		m[subTask.ID] = subTask
	}
	return m
}

// @llr REQ-TRAQ-SWL-9
func (r *Req) Changelists() map[string]string {
	m := map[string]string{}
	if r.Level == config.LOW {
		var paths []string
		for _, c := range r.Children {
			paths = append(paths, c.Path)
		}
		urls := changelistUrlsForFilepaths(paths)
		for _, url := range urls {
			fields := strings.Split(url, "/")
			m[fields[len(fields)-1]] = url
		}
	}
	return m
}

func changelistUrlsForFilepaths(filepaths []string) []string {
	var urls []string
	for _, path := range filepaths {
		urls = append(urls, changelistUrlsForFilepath(path)...)
	}
	return urls
}

func changelistUrlsForFilepath(filepath string) []string {
	res, err := linepipes.All(linepipes.Run("git", "-C", path.Dir(filepath), "log", filepath))
	if err != nil {
		log.Fatal(err)
	}

	matches := reDiffRev.FindAllStringSubmatch(res, -1)
	if len(matches) < 1 {
		log.Printf("Could not extract differential revision for file: %s", filepath)
		log.Println("Newly added?")
	}

	var urls []string
	for _, m := range matches {
		if len(m) != 2 {
			log.Fatal("Count not extract changelist substring for filepath: ", filepath)
		}
		urls = append(urls, m[1])
	}

	return urls
}

// A ReqGraph maps IDs and Paths to Req structures.
// @llr REQ-TRAQ-SWL-15
type reqGraph map[string]*Req

func CreateReqGraph(certdocsPath, codePath string) (reqGraph, error) {
	rg := reqGraph{}
	errorResult := ""

	_ = filepath.Walk(filepath.Join(git.RepoPath(), certdocsPath),
		func(fileName string, info os.FileInfo, err error) error {
			var errs []error
			switch strings.ToLower(path.Ext(fileName)) {
			case ".md":
				errs = parseCertdocToGraph(fileName, rg)
			}
			if len(errs) > 0 {
				errorResult += "Problems found while parsing " + fileName + ":\n"
				for _, v := range errs {
					errorResult += "\t" + v.Error() + "\n"
				}
				errorResult += "\n"
			}
			return nil
		})

	// walk the code
	_ = filepath.Walk(filepath.Join(git.RepoPath(), codePath), func(fileName string, info os.FileInfo, err error) error {
		switch strings.ToLower(path.Ext(fileName)) {
		case ".cc", ".c", ".h", ".hh", ".go":
			// TODO (pk,lb): do that in a nicer way without hard-coded folder names
			if strings.Contains(codePath, "testdata") || !strings.Contains(fileName, "testdata") {
				id := relativePathToRepo(fileName, git.RepoPath())
				if id == "" {
					log.Fatal("Malformed code file path")
				}
				err = parseCode(id, fileName, rg)
				if err != nil {
					errorResult += err.Error()
					errorResult += "\n"
				}
			}
		}
		return nil
	})

	err := rg.Resolve()
	if err != nil {
		errorResult += err.Error()
	}

	if errorResult != "" {
		return rg, fmt.Errorf(errorResult)
	}
	return rg, nil
}

// relativePathToRepo returns filePath relative to repoPath by
// removing the path to the repository from filePath
func relativePathToRepo(filePath, repoPath string) string {
	fields := strings.SplitAfterN(filePath, repoPath, 2)
	if len(fields) < 2 {
		return ""
	}
	return fields[1][1:] // omit leading slash
}

func (rg reqGraph) AddReq(req *Req, path string) error {
	if v := rg[req.ID]; v != nil {
		return fmt.Errorf("Requirement %s in %s already defined in %s", req.ID, path, v.Path)
	}
	req.Path = strings.TrimPrefix(path, git.RepoPath())

	rg[req.ID] = req
	return nil
}

func (rg reqGraph) CheckAttributes(reportConf JsonConf, filter ReqFilter, diffs map[string][]string) ([]error, error) {
	var errs []error
	for _, req := range rg {
		if req.Level != config.CODE && req.Matches(filter, diffs) {
			errs = append(errs, req.CheckAttributes(reportConf.Attributes)...)
		}
	}
	return errs, nil
}

// @llr REQ-TRAQ-SWL-4
func (rg reqGraph) checkReqReferences(certdocPath string) ([]error, error) {
	reParents := regexp.MustCompile(`Parents: REQ-`)

	errors := make([]error, 0)

	err := filepath.Walk(filepath.Join(git.RepoPath(), certdocPath),
		func(fileName string, info os.FileInfo, err error) error {
			r, err := os.Open(fileName)
			if err != nil {
				return err
			}

			scan := bufio.NewScanner(r)
			for lno := 1; scan.Scan(); lno++ {
				line := scan.Text()
				// parents have alreay been checked in Resolve(), and we don't throw an eror at the place where the deleted req is defined
				discardRefToDeleted := reParents.MatchString(line) || ReReqDeleted.MatchString(line)
				parmatch := ReReqID.FindAllStringSubmatchIndex(line, -1)
				for _, ids := range parmatch {
					reqID := line[ids[0]:ids[1]]
					v, reqFound := rg[reqID]
					if !reqFound {
						errors = append(errors, fmt.Errorf("Invalid reference to inexistent requirement %s in %s:%d", reqID, fileName, lno))
					} else if v.IsDeleted() && !discardRefToDeleted {
						errors = append(errors, fmt.Errorf("Invalid reference to deleted requirement %s in %s:%d", reqID, fileName, lno))
					}
				}
			}
			return nil
		})

	if err != nil {
		return nil, err
	}

	return errors, nil
}

func (rg reqGraph) AddCodeRefs(id, fileName, fileHash string, reqIds []string) {
	rg[fileName] = &Req{ID: id, Path: fileName, FileHash: fileHash, ParentIds: reqIds, Level: config.CODE}
}

// @llr REQ-TRAQ-SWL-17
func (rg reqGraph) Resolve() error {
	errorResult := ""

	for _, req := range rg {
		if len(req.ParentIds) == 0 && !(req.Level == config.SYSTEM || req.IsDeleted()) {
			errorResult += "Requirement " + req.ID + " in file " + req.Path + " has no parents.\n"
		}
		for _, parentID := range req.ParentIds {
			parent := rg[parentID]
			if parent != nil {
				if parent.IsDeleted() && !req.IsDeleted() {
					if req.Level != config.CODE {
						errorResult += "Invalid parent of requirement " + req.ID + ": " + parentID + " is deleted.\n"
					} else {
						errorResult += "Invalid reference in file " + req.Path + ": " + parentID + " is deleted.\n"
					}
				}
				parent.Children = append(parent.Children, req)
				req.Parents = append(req.Parents, parent)
			} else {
				if req.Level != config.CODE {
					errorResult += "Invalid parent of requirement " + req.ID + ": " + parentID + " does not exist.\n"
				} else {
					errorResult += "Invalid reference in file " + req.Path + ": " + parentID + " does not exist.\n"
				}
			}
		}
	}

	if errorResult != "" {
		errorResult += "\n"
		return fmt.Errorf(errorResult)
	}

	for _, req := range rg {
		if req.Level == config.SYSTEM {
			req.resolveDown()
		}
	}

	for _, req := range rg {
		sort.Sort(byPosition(req.Parents))
		sort.Sort(byPosition(req.Children))
	}

	for _, req := range rg {
		if req.Level == config.CODE {
			req.resolveUp()
			req.Position = req.Parents[0].Position
		}
	}
	return nil
}

func (rg reqGraph) OrdsByPosition() []*Req {
	var r []*Req
	for _, v := range rg {
		if v.Level == config.SYSTEM {
			r = append(r, v)
		}
	}
	sort.Sort(byPosition(r))
	return r
}

func (rg reqGraph) CodeFilesByPosition() []*Req {
	var r []*Req
	for _, v := range rg {
		if v.Level == config.CODE {
			r = append(r, v)
		}
	}
	sort.Sort(byPosition(r))
	return r
}

// Updates the tasks associated with each requirement.For each requirement in rg, the method will:
// - find the task associated with the requirement, by searching for the requirement ID in the task title using the taskmgr API
// - if a task was found and the requirement was not deleted, its title and description are updated
// - if a task was found and the requirement was deleted, the task is set as INVALID
// - if the task was not found, it is created and filled in with the following values:
// 	Title: <Req ID> <Req Title>
//	Description: <Requirement Body>
//	Status: Open
//	Tags: Project Abbreviation (e.g. DDLN, VXU, etc.)
//      Parents: the first parent task (Phabricator doesn't yet support multiple parents in the api)
// The method performs a breadth-first search of the requirement graph, which ensures that all parent tasks have already
// been created by the time a child is visited.
func (rg reqGraph) UpdateTasks(filterIDs map[string]bool) error {
	queue := rg.OrdsByPosition()  // breadth-first traversal queue
	enqueued := map[string]bool{} // set of elements that have already been enqueued for traversal
	reqIDToTaskPHID := map[string]string{}
	const projectNameSYS = config.ProjectName + "-SYS"
	const projectNameHLR = config.ProjectName + "-HLR"
	const projectNameLLR = config.ProjectName
	sysProjectID, err := taskmgr.TaskMgr.GetOrCreateProject(projectNameSYS, "")
	if err != nil {
		return err
	}

	hlrsProjectID, err := taskmgr.TaskMgr.GetOrCreateProject(projectNameHLR, sysProjectID)
	if err != nil {
		return err
	}

	llrsProjectID, err := taskmgr.TaskMgr.GetOrCreateProject(config.ProjectName, hlrsProjectID)
	if err != nil {
		return err
	}

	parentTaskTitle := "Implement " + config.ProjectName
	parentOfAll, err := taskmgr.TaskMgr.FindTaskByTitle(parentTaskTitle, sysProjectID)
	if err != nil {
		return err
	}
	var parentOfAllPHID string
	if parentOfAll == nil {
		log.Printf("Creating parent of all requirements: '%s'", parentTaskTitle)

		parentOfAllPHID, err = taskmgr.TaskMgr.CreateTask(parentTaskTitle, "Meta-task that incorporates all tasks needed to implement "+config.ProjectName,
			sysProjectID, map[string]string{}, []string{})
		if err != nil {
			return fmt.Errorf("Error creating parent of all tasks, %v", err)
		}
	} else {
		parentOfAllPHID = parentOfAll.ID
	}

	taskLevelToProjectPHID := map[config.RequirementLevel]string{config.SYSTEM: sysProjectID, config.HIGH: hlrsProjectID, config.LOW: llrsProjectID}
	for len(queue) > 0 {
		currentReq := queue[0]
		queue = queue[1:]
		if currentReq.Level == config.CODE {
			continue
		}
		projectPHID := taskLevelToProjectPHID[currentReq.Level]
		task, err := taskmgr.TaskMgr.FindTask(currentReq.ID, currentReq.Title, projectPHID)
		if err != nil {
			return fmt.Errorf("Error finding task for requirement %s, caused by\n%v", currentReq.ID, err)
		}

		var parentTaskIDs []string

		if currentReq.Level == config.SYSTEM {
			parentTaskIDs = []string{parentOfAllPHID}
		} else { // HLR or LLR
			for _, parentReq := range currentReq.Parents {
				taskID, ok := reqIDToTaskPHID[parentReq.ID]
				if !ok {
					return fmt.Errorf("Error updating requirement %s. Parent %s has no corresponding task", currentReq.ID, parentReq.ID)
				}
				parentTaskIDs = append(parentTaskIDs, taskID)
			}
		}
		//TODO: add support for deleted tasks
		if filterIDs[currentReq.ID] { // don't update requirements that are filtered
			if task == nil {
				if !currentReq.IsDeleted() {
					log.Printf("Creating task for requirement %s", currentReq.ID)

					taskPHID, err := taskmgr.TaskMgr.CreateTask(currentReq.ID+": "+currentReq.Title, string(currentReq.Body),
						projectPHID, currentReq.Attributes, parentTaskIDs)
					if err != nil {
						return fmt.Errorf("Error creating requirement %s, caused by\n%v", currentReq.ID, err)
					}
					reqIDToTaskPHID[currentReq.ID] = taskPHID
				}
			} else {
				if currentReq.IsDeleted() {
					if task.Status != "invalid" {
						log.Printf("Marking task T%s for DELETED requirement %s as invalid", task.ID, currentReq.ID)

						err = taskmgr.TaskMgr.DeleteTask(task.ID, currentReq.ID+": "+currentReq.Title, projectPHID)
						if err != nil {
							return fmt.Errorf("Error updating requirement %s, caused by\n%v", currentReq.ID, err)
						}
					}
				} else {
					log.Printf("Updating task T%s for requirement %s", task.ID, currentReq.ID)
					err = taskmgr.TaskMgr.UpdateTask(task.ID, currentReq.ID+": "+currentReq.Title, string(currentReq.Body),
						projectPHID, currentReq.Attributes, parentTaskIDs)
					if err != nil {
						return fmt.Errorf("Error updating requirement %s, caused by\n%v", currentReq.ID, err)
					}
				}
			}
		}
		if task != nil {
			reqIDToTaskPHID[currentReq.ID] = task.ID
		}
		for _, childReq := range currentReq.Children {
			if _, ok := enqueued[childReq.ID]; !ok {
				enqueued[childReq.ID] = true
				queue = append(queue, childReq)
			}
		}
	}
	return nil
}

func (rg reqGraph) DanglingReqsByPosition() []*Req {
	var r []*Req
	for _, reg := range rg {
		if !reg.Seen {
			r = append(r, reg)
		}
	}
	sort.Sort(byPosition(r))
	return r
}

func (rg reqGraph) ReqsWithInvalidRequirementsByPosition() []*Req {
	var r []*Req

	return r
}

type byPosition []*Req

func (a byPosition) Len() int           { return len(a) }
func (a byPosition) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byPosition) Less(i, j int) bool { return a[i].Position < a[j].Position }

var reLLRReference = regexp.MustCompile(fmt.Sprintf(`//\s*@llr\s*(%s).*`, reReqIdStr))

func parseCode(id, fileName string, graph reqGraph) error {
	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	var refs []string
	h := sha1.New()
	// git compatible hash
	if s, err := f.Stat(); err == nil {
		fmt.Fprintf(h, "blob %d", s.Size())
		h.Write([]byte{0})
	}

	scanner := bufio.NewScanner(io.TeeReader(f, h))
	for scanner.Scan() {
		if parts := reLLRReference.FindStringSubmatch(scanner.Text()); len(parts) > 0 {
			refs = append(refs, parts[1])
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if len(refs) > 0 {
		graph.AddCodeRefs(id, fileName, string(h.Sum(nil)), refs)
	}
	return nil
}

func parseCertdocToGraph(fileName string, graph reqGraph) []error {
	reqs, err := ParseCertdoc(fileName)
	if err != nil {
		return []error{fmt.Errorf("Error parsing %s: %v", fileName, err)}
	}
	isReqPresent := make([]bool, len(reqs))

	var errs []error
	for i, v := range reqs {
		r, err := ParseReq(v)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		errs2 := lintReq(fileName, len(reqs), isReqPresent, r)
		if len(errs2) != 0 {
			errs = append(errs, errs2...)
			continue
		}
		r.Position = i
		graph.AddReq(r, fileName)
	}

	return errs
}

// lintReq is called for each requirement while building the req graph
// @llr REQ-TRAQ-SWL-3
// @llr REQ-TRAQ-SWL-5
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
	reqIDComps := strings.Split(r.ID, "-") // results in an array such as [REQ PROJECT REQTYPE 1234]
	// check requirement name
	if reqIDComps[0] != "REQ" {
		errs = append(errs, fmt.Errorf("Incorrect requirement name %s. Every requirement needs to start with REQ, got %s.", r.ID, reqIDComps[0]))
	}
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
		// check requirement sequence number
		if currentID > nReqs {
			errs = append(errs, fmt.Errorf("Invalid requirement sequence number for %s: missing requirements in between. Total number of requirements is %d.", r.ID, nReqs))
		} else {
			if currentID < 1 {
				errs = append(errs, fmt.Errorf("Invalid requirement sequence number for %s: first requirement has to start with 001.", r.ID))
			} else {
				if isReqPresent[currentID-1] {
					errs = append(errs, fmt.Errorf("Invalid requirement sequence number for %s, is duplicate.", r.ID))
				}
				isReqPresent[currentID-1] = true
			}
		}
	}

	return errs
}

type FilterType int

const (
	TitleFilter FilterType = iota
	IdFilter
	BodyFilter
)

type ReqFilter map[FilterType]*regexp.Regexp

// Matches returns true if the requirement matches the filter AND its ID is
// in the diffs map, if any.
// @llr REQ-TRAQ-SWL-12
func (r *Req) Matches(filter ReqFilter, diffs map[string][]string) bool {
	for t, e := range filter {
		switch t {
		case TitleFilter:
			if !e.MatchString(r.Title) {
				return false
			}
		case IdFilter:
			if !e.MatchString(r.ID) {
				return false
			}
		case BodyFilter:
			if !e.MatchString(string(r.Body)) {
				return false
			}
		}
	}
	if diffs == nil {
		return true
	}
	_, ok := diffs[r.ID]
	return ok
}

func NextId(f string) (string, error) {
	var (
		reqs      []string
		nextReqID string
	)

	reqs, err := ParseCertdoc(f)
	if err != nil {
		return "", err
	}

	if len(reqs) > 0 {
		var (
			lastReq    *Req
			greatestID int = 0
		)
		// infer next req ID from existing req IDs
		for _, v := range reqs {
			r, err := ParseReq(v)
			if err != nil {
				return "", err
			}
			parts := ReReqID.FindStringSubmatch(r.ID)
			if parts == nil {
				return "", fmt.Errorf("Requirement ID invalid: %s", r.ID)
			}
			sequenceNumber := parts[len(parts)-1]
			currentID, err := strconv.Atoi(sequenceNumber)
			if err != nil {
				return "", fmt.Errorf("Requirement sequence part \"%s\" (%s) not a number:  %s", r.ID, sequenceNumber, err)
			}
			if currentID > greatestID {
				greatestID = currentID
				lastReq = r
			}
		}
		ii := ReReqID.FindStringSubmatchIndex(lastReq.ID)
		nextReqID = fmt.Sprintf("%s%d", lastReq.ID[:ii[len(ii)-2]], greatestID+1)
	} else {
		// infer next (=first) req ID from file name
		fNameWithExt := path.Base(f)
		extension := filepath.Ext(fNameWithExt)
		fName := fNameWithExt[0 : len(fNameWithExt)-len(extension)]
		fNameComps := strings.Split(fName, "-")
		docType := fNameComps[len(fNameComps)-1]
		reqType, correctFileType := config.DocTypeToReqType[docType]
		if !correctFileType {
			return "", fmt.Errorf("Document name does not comply with naming convention.")
		}
		nextReqID = "REQ-" + fNameComps[0] + "-" + fNameComps[1] + "-" + reqType + "-001"
	}

	return nextReqID, nil
}

// ParseCertdoc parses raw requirements out of a certdoc.
func ParseCertdoc(fileName string) ([]string, error) {
	if err := IsValidDocName(fileName); err != nil {
		return nil, err
	}
	return ParseMarkdown(fileName)
}

// IsValidDocName checks the f filename is a valid certdoc name.
// @llr REQ-TRAQ-SWL-20
func IsValidDocName(f string) error {
	ext := path.Ext(f)
	if strings.ToLower(ext) != ".md" {
		return fmt.Errorf("Invalid extension: '%s'. Only '.md' is supported", strings.ToLower(ext))
	}
	filename := strings.TrimSuffix(path.Base(f), ext)
	// check if the structure of the filename is correct
	parts := reCertdoc.FindStringSubmatch(filename)
	if parts == nil {
		return fmt.Errorf("Invalid file name: '%s'. Certification doc file name must match %v", filename, reCertdoc)
	}
	// check the document type code
	docType := parts[3]
	correctNumber, ok := config.DocTypeToDocId[docType]
	if !ok {
		return fmt.Errorf("Invalid document type: '%s'. Must be one of %v", docType, config.DocTypeToDocId)
	}
	// check the document type number
	docNumber := parts[2]
	if correctNumber != docNumber {
		return fmt.Errorf("Document number for type '%s' must be '%s', and not '%s'", docType, correctNumber, docNumber)
	}
	return nil
}
