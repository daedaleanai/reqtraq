// This file defines the interface for handling tasks and the generic Task object.
// We expect specific implementation of this interface to be marked with a build tag (see maniphest.go for more details)
package taskmgr

// Task represents a single task in Maniphest, JIRA, Bugzilla, etc.
type Task struct {
	ID string
	// This ID will be displayed in the reports; can be the same as ID
	DisplayID        string
	Status           string
	IsClosed         bool
	Priority         string
	Title            string
	Description      string
	URI              string
	DependsOnTaskIDs []string
}

type TaskManager interface {
	//Project management: generally, DO-178C tasks belong to a project. This can be a Phabricator PROJECT ID,
	//a Buganizer Project or whatever the equivalent is in Bugzilla or JIRA. Reqtraq associates each requirement
	//with a task, and each task is within a project. System level, high-level and low-level requirements each
	//have their own projects (but this can be easily adapted)

	// GetProject returns the id of the project with the given name or nil if the project doesn't exist.
	// This method is only used for optimizing calls to FindTask, so may be left unimplemented
	GetProject(name string) (string, error)

	// CreateProject creates a project with the given name and parent and returns its ID.
	CreateProject(name, parentID string) (string, error)

	// Gets the project with the given name and parent if it exists, creates it otherwise
	GetOrCreateProject(name, parentID string) (string, error)

	// FindTaskByID returns the  task with the given ID
	FindTaskByID(id string) (*Task, error)

	// FindTaskByTitle by title returns the task with the given title, nil if the task is not found, or an error if
	// there was an error finding the task. In case there are multiple tasks with the given title, FindTaskByTitle
	// returns an error.
	FindTaskByTitle(taskTitle, projectID string) (*Task, error)

	// FindTask returns the task corresponding to the given Requirement ID, nil if the task was not found or an error if
	// there was an error finding the task.
	// Note: The task title is usually "<reqid> reqname", so the uniqueness requirement is not unreasonable
	FindTask(requirementID, requirementTitle, projectID string) (*Task, error)

	// UpdateTask updates the task with the given ID with the data from the given parameters
	UpdateTask(taskID, title, taskBody, projectID string, attributes map[string]string, parentTaskIDs []string) error

	// DeleteTask closes the task with the given ID (or simply deletes the task if the task management tool supports
	// task deletion)
	DeleteTask(taskID, title, projectID string) error

	// CreateTask creates a new task with the given parameters
	CreateTask(title, taskBody, projectID string, attributes map[string]string, parentTaskIDs []string) (string, error)
}
