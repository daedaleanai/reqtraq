/*
Wrapper functions for git commands used within Reqtraq.
*/

package git

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/daedaleanai/reqtraq/linepipes"
)

var repoNames = make(map[string]string)

// RepoName returns the name for the current git repository (i.e. the repository for the current working directory)
// @llr REQ-TRAQ-SWL-16
func RepoName() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	path, ok := repoNames[cwd]
	if ok {
		return path
	}

	var name string
	// See details about "working directory" in https://git-scm.com/docs/githooks
	bare, err := linepipes.Single(linepipes.Run("git", "rev-parse", "--is-bare-repository"))
	if err != nil {
		log.Fatal(err)
	}
	if bare == "true" {
		// A bare repository is a dir identical in structure to the usual .git dir, but
		// never associated with a working tree.
		currentDir, err := os.Getwd()
		if err != nil {
			log.Fatalf("Failed to get current dir: %v", err)
		}
		name = filepath.Base(currentDir)
	} else {
		toplevel, err := linepipes.Single(linepipes.Run("git", "rev-parse", "--show-toplevel"))
		if err != nil {
			log.Fatal(err)
		}
		name = filepath.Base(toplevel)
	}
	name = strings.TrimSuffix(name, ".git")
	repoNames[cwd] = name
	return name
}

var repoPaths = make(map[string]string)

// RepoPath returns the full path of the current git repository's root.
// @llr REQ-TRAQ-SWL-16
func RepoPath() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	path, ok := repoPaths[cwd]
	if ok {
		return path
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
	repoPaths[cwd] = toplevel
	return toplevel
}

// AllCommits returns the list of commits formatted as "ID DATE".
// @llr REQ-TRAQ-SWL-16
func AllCommits() ([]string, error) {
	commits := make([]string, 0)
	lines, errs := linepipes.Run("git", "log", `--pretty=format:%h %cd`, "--date=short")
	for line := range lines {
		commits = append(commits, line)
	}
	if err := <-errs; err != nil {
		return commits, fmt.Errorf("Failed to get the list of commits: %s", err)
	}
	return commits, nil
}

// Clone clones the repo in a new temporary directory and returns it.
// @llr REQ-TRAQ-SWL-16
func Clone() (string, error) {
	repo := RepoPath()
	cloneDir, err := ioutil.TempDir("", "clone")
	if err != nil {
		return "", err
	}
	if err := os.Chdir(cloneDir); err != nil {
		return "", err
	}
	if _, err := linepipes.All(linepipes.Run("git", "clone", repo, ".")); err != nil {
		return "", err
	}
	return cloneDir, nil
}

// Checkout checks out the specified commit, branch, tag, etc.
// @llr REQ-TRAQ-SWL-16
func Checkout(commit string) error {
	_, err := linepipes.All(linepipes.Run("git", "checkout", commit))
	return err
}
