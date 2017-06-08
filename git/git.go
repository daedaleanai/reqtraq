//@llr REQ-0-DDLN-SWL-001
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
			log.Fatalf("Failed to get current dir: %r", err)
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

func CurrentBranch() (string, error) {
	return linepipes.Single(linepipes.Run("git", "rev-parse", "--abbrev-ref", "HEAD"))
}

func CheckIsAncestor(oldCommit string, newCommit string) error {
	return linepipes.Out(linepipes.Run("git", "merge-base", "--is-ancestor", oldCommit, newCommit))
}

func PathInRepo(localpath string) (string, error) {
	return linepipes.Single(linepipes.Run("git", "ls-tree", "--full-name", "--name-only", "HEAD", localpath))
}

func FilesChangedInIndex() ([]string, []string, error) {
	return FilesChanged("--cached")
}

func FilesChangedInCommit(commit string) ([]string, error) {
	lines, errors := linepipes.Run("git", "diff-tree", "--no-commit-id", "--name-only", "-r", commit)
	res := make([]string, 0)
	for line := range lines {
		res = append(res, line)
	}
	if err, _ := <-errors; err != nil {
		return res, fmt.Errorf("Failed to get changed files in commit: %s", err)
	}
	return res, nil
}

// FilesChangedBetween returns the paths of the files changed in a range of commits.
// The paths are relative to the repo root dir.
func FilesChangedBetween(commit1, commit2 string) ([]string, []string, error) {
	commitsRange := fmt.Sprintf("%s..%s", commit1, commit2)
	return FilesChanged(commitsRange)
}

func FilesChanged(args ...string) ([]string, []string, error) {
	args = append(append(make([]string, 0), "diff", "--name-status"), args...)
	lines, errors := linepipes.Run("git", args...)
	changedFiles := make([]string, 0)
	deletedFiles := make([]string, 0)
	for line := range lines {
		parts := strings.Split(line, "\t")
		switch parts[0][0] {
		case 'D': // Deleted
			deletedFiles = append(deletedFiles, parts[1])
		case 'A', 'M': // Added, Modified
			changedFiles = append(changedFiles, parts[1])
		case 'R': // Renamed
			deletedFiles = append(deletedFiles, parts[1])
			changedFiles = append(changedFiles, parts[2])
		case 'C': // Copied
			changedFiles = append(changedFiles, parts[2])
		case 'T': // have their type (i.e. regular file, symlink, submodule, etc) changed
			// Don't bother.
		default:
			// See --diff-filter in https://git-scm.com/docs/git-diff
			// Could be: Unmerged (U), Unknown (X), Broken pairing (B)
			return nil, nil, fmt.Errorf("Unexpected status: %s", line)
		}
	}
	if err, _ := <-errors; err != nil {
		return changedFiles, deletedFiles, fmt.Errorf("Failed to get changed files: %s", err)
	}
	return changedFiles, deletedFiles, nil
}

func FilesChangedOnMergedBranch(mergeCommit string) ([]string, []string, error) {
	previous := fmt.Sprintf("%s^1", mergeCommit)
	// The 2nd parent of the merge commit is the top of the branch merged into "master".
	merged := fmt.Sprintf("%s^2", mergeCommit)
	mergeBase, err := linepipes.Single(linepipes.Run("git", "merge-base", previous, merged))
	if err != nil {
		return nil, nil, err
	}
	return FilesChangedBetween(mergeBase, merged)
}

// AllCommits returns the list of commits formatted as "ID DATE".
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

func CommitsBetween(commit1, commit2 string) ([]string, error) {
	commits := make([]string, 0)
	lines, errs := linepipes.Run("git", "rev-list", commit2, "--not", commit1)
	for line := range lines {
		commits = append(commits, line)
	}
	if err := <-errs; err != nil {
		return commits, fmt.Errorf("Failed to get the list of new commits: %s", err)
	}
	return commits, nil
}

// Clone clones the repo in a new temporary directory and returns it.
func Clone() (string, error) {
	repo := RepoPath()
	cloneDir, err := ioutil.TempDir("", "clone")
	if err != nil {
		return "", err
	}
	if err := os.Chdir(cloneDir); err != nil {
		return "", err
	}
	if err := linepipes.Out(linepipes.Run("git", "clone", repo, ".")); err != nil {
		return "", err
	}
	return cloneDir, nil
}

// Checkout checks out the specified commit, branch, tag, etc.
func Checkout(commit string) error {
	return linepipes.Out(linepipes.Run("git", "checkout", commit))
}
