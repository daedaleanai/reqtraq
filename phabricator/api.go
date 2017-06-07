// @llr REQ-0-DDLN-SWL-018
package phabricator

import (
	"errors"
	"fmt"
	"strings"

	"github.com/arbovm/levenshtein"
	"github.com/danieldanciu/gonduit"
	"github.com/danieldanciu/gonduit/core"
	"github.com/danieldanciu/gonduit/entities"
	"github.com/danieldanciu/gonduit/requests"

	"go.daedalean.ai/exp-devtools/linepipes"
)

var cachedApiToken string

// getApiToken returns the Phabricator API token that allows us to make authenticated API operations.
func getApiToken() (string, error) {
	if cachedApiToken != "" {
		return cachedApiToken, nil
	}

	// The Phabricator API token comes from https://p.daedalean.ai/settings/user/git/page/apitokens/ (the already
	// generated Standard Api Token is good to use
	//
	// set it on the git server repository like this:
	// $ ssh git.daedalean.ai
	// $ cd /var/git/exp.git
	// $ sudo -u git sh -c "git config --local --replace-all daedalean.phabricator-api-token <TOKEN>"
	apiToken, err := linepipes.Single(linepipes.Run("git", "config", "--get", "daedalean.phabricator-api-token"))
	if err != nil {
		msg := `No Phabricator API token set. Please go to
	https://p.daedalean.ai/settings/user/<YOUR_USERNAME_HERE>/page/apitokens/
click on <Generate API Token>, and then paste the token into this command
	git config --local --replace-all daedalean.phabricator-api-token <PASTE_TOKEN_HERE>`
		return "", fmt.Errorf(msg)
	}
	cachedApiToken = apiToken
	return apiToken, nil
}

// getApiClient returns a client connection to the Phabricator server
func getApiClient() (*gonduit.Conn, error) {
	apiToken, err := getApiToken()
	if err != nil {
		return nil, err
	}
	return gonduit.Dial("https://p.daedalean.ai", &core.ClientOptions{APIToken: apiToken})
}

// GetRevision returns the Differential revision with the given ID or an error if the revision is not found
func GetRevision(revisionID uint64) (*entities.DifferentialRevision, error) {
	client, err := getApiClient()
	if err != nil {
		return nil, err
	}
	res, err := client.DifferentialQuery(requests.DifferentialQueryRequest{IDs: []uint64{revisionID}})
	if err != nil {
		return nil, err
	}
	if len(*res) == 0 {
		return nil, errors.New("Cannot be found")
	}
	return (*res)[0], nil
}

// GetProject returns the PHID of the Phabricator project with the given name or nil if the project doesn't exist.
// For example, when called with "Reqtraq" the method will return "PHID-PROJ-3e2qnmcuzxzl3iko7xdl".
// The method calls https://p.daedalean.ai/api/project.query with the request
//	{
//		"names": ["Reqtraq"]
//	}
func GetProject(name string) (*entities.Project, error) {
	client, err := getApiClient()
	if err != nil {
		return nil, err
	}
	res, err := client.ProjectQuery(requests.ProjectQueryRequest{Names: []string{name}})
	if err != nil {
		return nil, err
	}
	if len(res.Data) == 0 {
		return nil, nil
	}
	if len(res.Data) > 1 {
		return nil, fmt.Errorf("Ambiguous project name: %s. Multiple projects match the name", name)
	}
	for _, v := range res.Data {
		return &v, nil
	}
	return nil, nil
}

// CreateProject creates a Phabricator project with the given name and parent and returns it.
func CreateProject(name, parentPHID string) (*entities.Project, error) {
	client, err := getApiClient()
	if err != nil {
		return nil, err
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
		return nil, err
	}

	return project, nil
}

func GetOrCreateProject(name, parentPHID string) (*entities.Project, error) {
	project, err := GetProject(name)
	if err != nil {
		return nil, err
	}
	if project != nil {
		return project, nil
	}
	return CreateProject(name, parentPHID)
}

// FindTaskByPHID returns the Phabricator task with the given PHID string
func FindTaskByPHID(phid string) (*entities.ManiphestTask, error) {
	client, err := getApiClient()
	if err != nil {
		return nil, err
	}

	res, err := client.ManiphestQuery(requests.ManiphestQueryRequest{PHIDs: []string{phid}})
	if err != nil {
		return nil, err
	}

	for _, v := range *res {
		return v, nil
	}
	return nil, fmt.Errorf("No task with phid %s found.", phid)
}

// FindTaskByTitle by title returns the Maniphest task with the given title, nil if the task is not found, or an error if
// there was an error finding the task. In case there are multiple tasks with the given title, FindTaskByTitle
// returns an error.
func FindTaskByTitle(taskTitle, projectPHID string) (*entities.ManiphestTask, error) {
	client, err := getApiClient()
	if err != nil {
		return nil, err
	}
	res, err := client.ManiphestQuery(requests.ManiphestQueryRequest{FullText: taskTitle, ProjectPHIDs: []string{projectPHID}})
	if err != nil {
		return nil, err
	}

	var tasks []*entities.ManiphestTask
	for _, task := range *res {
		if task.Title == taskTitle {
			tasks = append(tasks, task)
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
func FindTask(requirementID, requirementTitle, projectPHID string) (*entities.ManiphestTask, error) {
	client, err := getApiClient()
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
		return tasks[0], nil
	}
	bestTask := tasks[0]
	bestDist := levenshtein.Distance(bestTask.Title, requirementTitle)
	for _, task := range tasks {
		if currentDist := levenshtein.Distance(task.Title, requirementTitle); currentDist < bestDist {
			bestDist = currentDist
			bestTask = task
		}
	}
	return bestTask, nil
}

// UpdateTask updates the Maniphest task with the given ID with the data from the given parameters
func UpdateTask(taskID, title, taskBody, projectPHID string, attributes map[string]string, parentTaskIDs []string) error {
	client, err := getApiClient()
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
func DeleteTask(taskID, title, projectPHID string) error {
	client, err := getApiClient()
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

// UpdateTask updates the Maniphest task with the given ID with the data from the given map
func CreateTask(title, taskBody, projectPHID string, attributes map[string]string, parentTaskIDs []string) (string, error) {
	client, err := getApiClient()
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
