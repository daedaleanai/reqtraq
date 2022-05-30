package repos

import (
	"fmt"
	"io/ioutil"
	"io/fs"
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

// Maps from name to path
var repositories map[RepoName]RepoPath = make(map[RepoName]RepoPath)

// Names from remote paths are assumed to be of either of these forms:
func GetRepoNameFromRemotePath(remote_path RemotePath) RepoName {
	splits := strings.Split(string(remote_path), "/")
	name := splits[len(splits) - 1]

	// If name ends with ".git" strip it, as that is part of the git URL
	name = strings.TrimSuffix(name, ".git")
	return RepoName(name)
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

// If it is already registered, it just returns the key and path as stored internally
func GetRepoPathByRemotePath(remote_path RemotePath) (RepoName, RepoPath, error) {
	// Deduce the name out of the repository
	name := GetRepoNameFromRemotePath(remote_path)

	// Check if it is already registered, if so just return it!
	repoPath, err := GetRepoPathByName(name)
	if err == nil {
		return name, repoPath, nil
	}

	// Clone the repo
	path, err := cloneFromRemote(remote_path);
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
func cloneFromRemote(remotePath RemotePath) (RepoPath, error) {
	cloneDir, err := ioutil.TempDir("", ".reqtraq")
	if err != nil {
		return "", err
	}
	if err := os.Chdir(cloneDir); err != nil {
		return "", err
	}
	if _, err := linepipes.All(linepipes.Run("git", "clone", string(remotePath))); err != nil {
		return "", err
	}
	return RepoPath(fmt.Sprintf("%s/%s", cloneDir, GetRepoNameFromRemotePath(remotePath))), nil
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

	err = filepath.Walk(actualPath, func (path string, fileInfo fs.FileInfo, err error) error {
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

func ValidatePath(repoName RepoName, path string) error {
	repoPath, err := GetRepoPathByName(repoName)
	if err != nil {
		return err
	}
	actualPath := fmt.Sprintf("%s/%s", repoPath, path)

	if _, err := os.Stat(actualPath); err == nil {
		return nil
	} else {
		return fmt.Errorf("Path `%s` does not seem to be accessible from the user: %s", actualPath, err)
	}
}
