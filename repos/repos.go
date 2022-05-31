package repos

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/daedaleanai/reqtraq/linepipes"
)

// A reference to a remote repository that can be one of:
//   - An HTTP url referencing a remote git repository
//   - A ssh url referencing a remote git repository
//   - A git url referencing a remote git repository
//   - A local repository that will be cloned
type RemotePath string

// Identifies a single repository
type RepoName string

// A path to a local repository that is present in the current filesystem
type RepoPath string

var (
	basePath string   = ""
	baseName RepoName = RepoName("")
	tempDirs []string = make([]string, 0)
)

// Maps from name to path
var repositories map[RepoName]RepoPath = make(map[RepoName]RepoPath)

// Names from remote paths are assumed to be of either of these forms:
func GetRepoNameFromRemotePath(remotePath RemotePath) RepoName {
	splits := strings.Split(string(remotePath), "/")
	name := splits[len(splits)-1]

	// If name ends with ".git" strip it, as that is part of the git URL
	name = strings.TrimSuffix(name, ".git")
	return RepoName(name)
}

func BaseRepoPath() string {
	if basePath != "" {
		return basePath
	}

	// See details about "working directory" in https://git-scm.com/docs/githooks
	bare, err := linepipes.Single(linepipes.Run("git", "rev-parse", "--is-bare-repository"))
	if err != nil {
		log.Fatal("Failed to check Git repository type. Are you running reqtraq in a Git repo?\n", err)
	}
	if bare == "true" {
		log.Fatal("Bare repository.")
	}

	toplevel, err := linepipes.Single(linepipes.Run("git", "rev-parse", "--show-toplevel"))
	if err != nil {
		log.Fatal(err)
	}

	basePath = toplevel
	baseName = GetRepoNameFromRemotePath(RemotePath(toplevel))
	return basePath
}

func BaseRepoName() RepoName {
	return baseName
}

func RegisterCurrentRepository(path string) RepoName {
	// Name is deduced from path
	name := GetRepoNameFromRemotePath(RemotePath(path))
	repositories[name] = RepoPath(path)

	return name
}

func ClearAllRepositories() {
	repositories = make(map[RepoName]RepoPath)
}

// Checks out the given git reference for the given repository. Creates a new clone of the repo to
// ensure not to alter the original repository. The repository must already be registered
func OverrideRepository(remotePath RemotePath, gitReference string) error {
	repoName := GetRepoNameFromRemotePath(remotePath)
	originalRepoPath, err := GetRepoPathByName(repoName)
	if err != nil {
		return err
	}

	// Clone the repo
	path, err := cloneFromRemote(RemotePath(originalRepoPath), gitReference)
	if err != nil {
		return err
	}

	// Now let's override it
	repositories[repoName] = path
	return nil
}

// If it is already registered, it just returns the key and path as stored internally
func GetRepoPathByRemotePath(remotePath RemotePath) (RepoName, RepoPath, error) {
	// Deduce the name out of the repository
	name := GetRepoNameFromRemotePath(remotePath)

	// Check if it is already registered, if so just return it!
	repoPath, err := GetRepoPathByName(name)
	if err == nil {
		return name, repoPath, nil
	}

	// Clone the repo
	path, err := cloneFromRemote(remotePath, "")
	if err != nil {
		return "", "", err
	}

	// Now let's store it
	repositories[name] = path
	return name, path, nil
}

// Obtains the path of a repository from its name
func GetRepoPathByName(name RepoName) (RepoPath, error) {
	if path, ok := repositories[name]; ok {
		return path, nil
	}
	return "", fmt.Errorf("Could not find path for repository with name `%s`", name)
}

/// Obtains a RepoPath from a RemotePath by cloning the repository locally
func cloneFromRemote(remotePath RemotePath, gitReference string) (RepoPath, error) {
	cloneDir, err := ioutil.TempDir("", ".reqtraq")
	if err != nil {
		return "", err
	}

	repoPath := RepoPath(filepath.Join(cloneDir, string(GetRepoNameFromRemotePath(remotePath))))

	if _, err := linepipes.All(linepipes.Run("git", "clone", string(remotePath), string(repoPath))); err != nil {
		return "", err
	}

	if gitReference != "" {
		originalCwd, err := os.Getwd()
		if err != nil {
			panic("Could not obtain current working directory!")
		}
		if err := os.Chdir(string(repoPath)); err != nil {
			return "", err
		}
		if _, err := linepipes.All(linepipes.Run("git", "checkout", gitReference)); err != nil {
			return "", err
		}
		if err := os.Chdir(originalCwd); err != nil {
			return "", err
		}
	}

	// Save the  temp dir for cleanup when we exit
	tempDirs = append(tempDirs, string(repoPath))
	return repoPath, nil
}

func CleanupTemporaryDirectories() {
	for _, dir := range tempDirs {
		os.RemoveAll(dir)
	}
}

// 1. Repo where files are located
// 2. The path to look in
// 3. pattern to match agains
// 4. Ignored paths
func FindFilesInDirectory(repoName RepoName, path string, pattern *regexp.Regexp, ignoredPaths []*regexp.Regexp) ([]string, error) {
	var files []string

	repoPath, err := GetRepoPathByName(repoName)
	if err != nil {
		return []string{}, err
	}
	actualPath := filepath.Join(string(repoPath), path)

	err = filepath.Walk(actualPath, func(path string, fileInfo fs.FileInfo, err error) error {
		// First lets start by removing the prefix from the actualPath
		relativePath, err := filepath.Rel(string(repoPath), path)
		if err != nil {
			return fmt.Errorf(`Error while walking a path and removing the prefix.
This should not happen. Please inform the developers by rasing an issue if you see this.`)
		}

		// Match path against ignoredPaths. If it does match, return skipdir
		for _, ignoredPath := range ignoredPaths {
			if ignoredPath.MatchString(relativePath) {
				return nil
			}
		}

		if fileInfo == nil || fileInfo.IsDir() {
			// We do not add directories to the list
			return nil
		}

		// Match path against pattern. If it does not match, continue
		if pattern != nil {
			if match := pattern.MatchString(relativePath); !match {
				// Does not match pattern, filter it out
				return nil
			}
		}

		// For everything else, append it
		files = append(files, relativePath)

		return nil
	})

	if err != nil {
		return []string{}, err
	}

	return files, nil
}

func PathInRepo(repoName RepoName, path string) (string, error) {
	repoPath, err := GetRepoPathByName(repoName)
	if err != nil {
		return "", err
	}
	actualPath := fmt.Sprintf("%s/%s", repoPath, path)

	return actualPath, nil
}

func ValidatePath(repoName RepoName, path string) error {
	repoPath, err := PathInRepo(repoName, path)
	if err != nil {
		return err
	}
	if _, err := os.Stat(repoPath); err == nil {
		return nil
	} else {
		return fmt.Errorf("Path `%s` does not seem to be accessible from the user: %s", repoPath, err)
	}
}

// AllCommits returns the list of commits formatted as "ID DATE".
// @llr REQ-TRAQ-SWL-16
func AllCommits(repoName RepoName) ([]string, error) {
	repoPath, err := GetRepoPathByName(repoName)
	if err != nil {
		return []string{}, err
	}

	originalCwd, err := os.Getwd()
	if err != nil {
		panic("Could not obtain current working directory!")
	}
	if err := os.Chdir(string(repoPath)); err != nil {
		return []string{}, err
	}
	commits := make([]string, 0)
	lines, errs := linepipes.Run("git", "log", `--pretty=format:%h %cd`, "--date=short")

	if err := os.Chdir(originalCwd); err != nil {
		return []string{}, err
	}

	for line := range lines {
		commits = append(commits, line)
	}

	if err := <-errs; err != nil {
		return commits, fmt.Errorf("Failed to get the list of commits: %s", err)
	}
	return commits, nil
}
