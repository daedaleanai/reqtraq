// This file is very Phabricator/Daedalean specific and is marked so with the Phabricator build tag. If you want to use another task
// management system instead of Phabricator, simply copy this file, change the tag below to e.g. Bugzilla or JIRA and
// re-implement the TaskManager interface to your liking. Remove the !phabricator tag from this file and then do a
// conditional build using
// 	go build -tags <your_tag>

// +build phabricator !phabricator

// @llr REQ-TRAQ-SWL-018
package taskmgr

import (
	"fmt"
	"strings"

	"github.com/arbovm/levenshtein"
	"github.com/danieldanciu/gonduit"
	"github.com/danieldanciu/gonduit/core"
	"github.com/danieldanciu/gonduit/entities"
	"github.com/danieldanciu/gonduit/requests"

	"github.com/daedaleanai/reqtraq/linepipes"
)

type PhabricatorTaskManager struct {
	cachedApiToken string
}

var TaskMgr TaskManager = &PhabricatorTaskManager{}

// getApiToken returns the Phabricator API token that allows us to make authenticated API operations.
func (tmgr *PhabricatorTaskManager) getApiToken() (string, error) {
	if tmgr.cachedApiToken != "" {
		return tmgr.cachedApiToken, nil
	}

	// The Phabricator API token comes from https://p.daedalean.ai/settings/user/git/page/apitokens/ (the already
	// generated Standard Api Token is good to use
	//
	// set it on the git server repository like this:
	// $ ssh git.daedalean.ai
	// $ cd /var/git/exp.git
	// $ sudo -u git sh -c "git config --local --replace-all daedalean.taskmgr-api-token <TOKEN>"
	apiToken, err := linepipes.Single(linepipes.Run("git", "config", "--get", "daedalean.taskmgr-api-token"))
	if err != nil {
		msg := `No Phabricator API token set. Please go to
	https://p.daedalean.ai/settings/user/<YOUR_USERNAME_HERE>/page/apitokens/
click on <Generate API Token>, and then paste the token into this command
	git config --local --replace-all daedalean.taskmgr-api-token <PASTE_TOKEN_HERE>`
		return "", fmt.Errorf(msg)
	}
	tmgr.cachedApiToken = apiToken
	return apiToken, nil
}

// getApiClient returns a client connection to the Phabricator server
func (tmgr *PhabricatorTaskManager) getApiClient() (*gonduit.Conn, error) {
	apiToken, err := tmgr.getApiToken()
	if err != nil {
		return nil, err
	}
	return gonduit.Dial("https://p.daedalean.ai", &core.ClientOptions{APIToken: apiToken})
}

// GetProject returns the ID of the Phabricator project ID with the given name or nil if the project doesn't exist.
// For example, when called with "Reqtraq" the method will return "ID-PROJ-3e2qnmcuzxzl3iko7xdl".
// The method calls https://p.daedalean.ai/api/project.query with the request
//	{
//		"names": ["Reqtraq"]
//	}
func (tmgr *PhabricatorTaskManager) GetProject(name string) (string, error) {
	client, err := tmgr.getApiClient()
	if err != nil {
		return "", err
	}
	res, err := client.ProjectQuery(requests.ProjectQueryRequest{Names: []string{name}})
	if err != nil {
		return "", err
	}
	if len(res.Data) == 0 {
		return "", nil
	}
	if len(res.Data) > 1 {
		return "", fmt.Errorf("Ambiguous project name: %s. Multiple projects match the name", name)
	}
	for _, v := range res.Data {
		return v.PHID, nil
	}
	return "", nil
}

// CreateProject creates a Phabricator project with the given name and parent and returns its ID.
func (tmgr *PhabricatorTaskManager) CreateProject(name, parentPHID string) (string, error) {
	client, err := tmgr.getApiClient()
	if err != nil {
		return "", err
	}
	transactions := []requests.Transaction{
		requests.Transaction{TransactionType: "name", Value: name},
	}
	if parentPHID != "" {
		transactions = append(transactions, requests.Transaction{TransactionType: "parent", Value: parentPHID})
	}
	project, err := client.ProjectEdit(requests.EditEndpointRequest{
		Transactions: transactions})
	if err != nil {
		return "", err
	}

	return project.PHID, nil
}

func (tmgr *PhabricatorTaskManager) GetOrCreateProject(name, parentPHID string) (string, error) {
	projectID, err := tmgr.GetProject(name)
	if err != nil {
		return "", err
	}
	if projectID != "" {
		return projectID, nil
	}
	return tmgr.CreateProject(name, parentPHID)
}

// FindTaskByID returns the Phabricator task with the given ID string
func (tmgr *PhabricatorTaskManager) FindTaskByID(phid string) (*Task, error) {
	client, err := tmgr.getApiClient()
	if err != nil {
		return nil, err
	}

	res, err := client.ManiphestQuery(requests.ManiphestQueryRequest{PHIDs: []string{phid}})
	if err != nil {
		return nil, err
	}

	for _, v := range *res {
		return maniphestTaskToTask(v), nil
	}
	return nil, fmt.Errorf("No task with phid %s found.", phid)
}

// FindTaskByTitle by title returns the Maniphest task with the given title, nil if the task is not found, or an error if
// there was an error finding the task. In case there are multiple tasks with the given title, FindTaskByTitle
// returns an error.
func (tmgr *PhabricatorTaskManager) FindTaskByTitle(taskTitle, projectPHID string) (*Task, error) {
	client, err := tmgr.getApiClient()
	if err != nil {
		return nil, err
	}
	res, err := client.ManiphestQuery(requests.ManiphestQueryRequest{FullText: taskTitle, ProjectPHIDs: []string{projectPHID}})
	if err != nil {
		return nil, err
	}

	var tasks []*Task
	for _, task := range *res {
		if task.Title == taskTitle {
			tasks = append(tasks, maniphestTaskToTask(task))
		}
	}
	if len(tasks) == 0 {
		return nil, nil
	}
	if len(tasks) == 1 {
		return tasks[0], nil
	}
	return nil, fmt.Errorf("Multiple tasks found with title '%s'", taskTitle)
}

// FindTask returns the Maniphest task corresponding to the given Requirement ID, nil if the task was not found or an error if
// there was an error finding the task. In case there are multiple tasks with the given ID in the title, FindTask
// deterministically selects the task with the title that matches as closely as possible the given title
func (tmgr *PhabricatorTaskManager) FindTask(requirementID, requirementTitle, projectPHID string) (*Task, error) {
	client, err := tmgr.getApiClient()
	if err != nil {
		return nil, err
	}
	res, err := client.ManiphestQuery(requests.ManiphestQueryRequest{FullText: requirementID, ProjectPHIDs: []string{projectPHID}})
	if err != nil {
		return nil, err
	}

	var tasks []*entities.ManiphestTask
	for _, task := range *res {
		if strings.Contains(task.Title, requirementID) {
			tasks = append(tasks, task)
		}
	}
	if len(tasks) == 0 {
		return nil, nil
	}
	if len(tasks) == 1 {
		return maniphestTaskToTask(tasks[0]), nil
	}
	bestTask := tasks[0]
	bestDist := levenshtein.Distance(bestTask.Title, requirementTitle)
	for _, task := range tasks {
		if currentDist := levenshtein.Distance(task.Title, requirementTitle); currentDist < bestDist {
			bestDist = currentDist
			bestTask = task
		}
	}
	return maniphestTaskToTask(bestTask), nil
}

// UpdateTask updates the Maniphest task with the given ID with the data from the given parameters
func (tmgr *PhabricatorTaskManager) UpdateTask(taskID, title, taskBody, projectPHID string, attributes map[string]string, parentTaskIDs []string) error {
	client, err := tmgr.getApiClient()
	if err != nil {
		return err
	}
	transactions := []requests.Transaction{
		requests.Transaction{TransactionType: "title", Value: title},
		requests.Transaction{TransactionType: "description", Value: taskBody},
		requests.Transaction{TransactionType: "projects.add", Value: []string{projectPHID}},
	}
	if len(parentTaskIDs) > 0 {
		transactions = append(transactions, requests.Transaction{TransactionType: "parent", Value: parentTaskIDs[0]})
	}
	_, err = client.ManiphestEditTask(requests.EditEndpointRequest{
		ObjectIdentifier: taskID,
		Transactions:     transactions})
	return err

}

// DeleteTask closes the Maniphest task with the given ID as INVALID
func (tmgr *PhabricatorTaskManager) DeleteTask(taskID, title, projectPHID string) error {
	client, err := tmgr.getApiClient()
	if err != nil {
		return err
	}
	transactions := []requests.Transaction{
		requests.Transaction{TransactionType: "title", Value: title},
		requests.Transaction{TransactionType: "projects.add", Value: []string{projectPHID}},
		requests.Transaction{TransactionType: "status", Value: "invalid"},
	}
	_, err = client.ManiphestEditTask(requests.EditEndpointRequest{
		ObjectIdentifier: taskID,
		Transactions:     transactions})
	return err

}

// CreateTask creates a new task with the given parameters
func (tmgr *PhabricatorTaskManager) CreateTask(title, taskBody, projectPHID string, attributes map[string]string, parentTaskIDs []string) (string, error) {
	client, err := tmgr.getApiClient()
	if err != nil {
		return "", err
	}
	transactions := []requests.Transaction{
		requests.Transaction{TransactionType: "title", Value: title},
		requests.Transaction{TransactionType: "description", Value: taskBody},
		requests.Transaction{TransactionType: "status", Value: "open"},
		requests.Transaction{TransactionType: "projects.set", Value: []string{projectPHID}},
	}
	if len(parentTaskIDs) > 0 {
		transactions = append(transactions, requests.Transaction{TransactionType: "parent", Value: parentTaskIDs[0]})
	}
	res, err := client.ManiphestEditTask(requests.EditEndpointRequest{
		Transactions: transactions})
	if err != nil {
		return "", err
	}
	return res.Object.PHID, err

}

func maniphestTaskToTask(task *entities.ManiphestTask) *Task {
	return &Task{
		ID: task.PHID,
		DisplayID:               task.ID,
		Title:            task.Title,
		DependsOnTaskIDs: task.DependsOnTaskPHIDs,
		Description:      task.Description,
		IsClosed:         task.IsClosed,
		Priority:         task.Priority,
		Status:           task.Status,
		URI:              task.URI,
	}
}
