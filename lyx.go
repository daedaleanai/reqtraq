// @llr REQ-TRAQ-SWL-014
// @llr REQ-TRAQ-SWL-002
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/daedaleanai/reqtraq/config"
	"github.com/daedaleanai/reqtraq/git"
)

// lyxState is the information needed to keep around on a stack to parse the
// nested inset/layout structure of a .lyx file
type lyxState struct {
	lineNo  int    // line on which this was found
	element string // layout/inset/preamble/etc
	arg     string // first token after the begin_layout or begin_inset
}

// a lyxStack keeps track of the \begin_  \end_ pairs
type lyxStack []lyxState

func (s *lyxStack) push(lno int, line, arg string) {
	element := strings.SplitN(line[len(`\begin_`):], " ", 2)[0]
	*s = append(*s, lyxState{lno, element, arg})
}
func (s *lyxStack) pop(lno int, line string) error {
	element := strings.SplitN(line[len(`\end_`):], " ", 2)[0]
	top := s.top()
	if top.element != element {
		return fmt.Errorf("lyx file malformed: begin %s line %d ended by end %s line %d", top.element, top.lineNo, element, lno)
	}
	if len(*s) > 0 {
		*s = (*s)[:len(*s)-1]
	}
	return nil
}
func (s lyxStack) top() lyxState {
	if len(s) > 0 {
		return s[len(s)-1]
	}
	return lyxState{}
}

// inNoteLayout returns true when the current state stack top is 'Layout' inside an 'inset Note'
func (s lyxStack) inNoteLayout() bool {
	size := len(s)
	if size < 2 {
		return false
	}
	return s[size-2].element == "inset" && s[size-2].arg == "Note" && s[size-1].element == "layout"
}

// ParseLyx reads a .lyx file finding blocks of text bracketed by
// notes containing "req:"  ...  "/req".
// It returns a slice of strings with one element per req:/req block
// containing the text in layout blocks, skipping (hopefully) the inset data.
// or an error describing a problem parsing the lines.
// It linkifies the lyx file and writes it to the provided writer.
func ParseLyx(f string, w io.Writer) error {
	var state lyxStack
	r, err := os.Open(f)
	if err != nil {
		return err
	}
	scan := bufio.NewScanner(r)

	// Cache some info related to the git repo context.
	repo := git.RepoName()
	repoPath, err := filepath.Abs(git.RepoPath())
	if err != nil {
		return err
	}
	absPath, err := filepath.Abs(f)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(absPath, repoPath) {
		return fmt.Errorf("File %s (%s) not in repo: %s", f, absPath, repoPath)
	}
	pathInRepo := strings.TrimPrefix(absPath, repoPath)
	dirInRepo := filepath.Dir(pathInRepo)

	previousLine := ""
	for lno := 1; scan.Scan(); lno++ {
		outline := scan.Text()
		line := outline
		// An empty line means that a Lyx paragraph has ended.
		istext := line != "" && !strings.HasPrefix(line, `\`) && !strings.HasPrefix(line, `#`)
		fields := strings.Fields(line)
		arg := ""
		if len(fields) > 1 {
			arg = fields[1]
		}
		switch {
		case line == `\use_hyperref false`:
			// Required so the anchors end up in the PDF when converting.
			outline = `\use_hyperref true`

		case strings.HasPrefix(line, `\begin_layout`):
			state.push(lno, line, arg)

		case strings.HasPrefix(line, `\begin_inset`):
			state.push(lno, line, arg)

		case strings.HasPrefix(line, `\end_layout`):
			if err = state.pop(lno, line); err != nil {
				return err
			}

		case strings.HasPrefix(line, `\end_inset`):
			if err = state.pop(lno, line); err != nil {
				return err
			}

		case istext && state.top().element != "inset":
			count := len(ReReqID.FindAllString(previousLine, -1))
			countCurrent := len(ReReqID.FindAllString(line, -1))
			indexes := ReReqID.FindAllStringIndex(previousLine+line, -1)
			if count+countCurrent < len(indexes) {
				// There is a requirement ID which is split on two lines.
				// We move the entire requirement to the second line.
				line = previousLine[indexes[count][0]:] + line
				previousLine = previousLine[:indexes[count][0]]
			}
			if outline, err = linkify(line, repo, dirInRepo); err != nil {
				return fmt.Errorf("malformed requirement: cannot linkify ID on line %d: %q because: %s", lno, line, err)
			}
		}
		if lno > 1 {
			if _, err := w.Write([]byte(previousLine + "\n")); err != nil {
				return err
			}
		}
		previousLine = outline
	}
	if _, err := w.Write([]byte(previousLine + "\n")); err != nil {
		return err
	}

	if err := scan.Err(); err != nil {
		return err
	}

	return nil
}

func linkify(s, repo, dirInRepo string) (string, error) {
	parmatch := ReReqID.FindAllStringSubmatchIndex(s, -1)
	var res bytes.Buffer
	parsedTo := 0
	for _, ids := range parmatch {
		// For example: ["REQ-TRAQ-SYS-006" "TRAQ" "SYS" "006"]
		res.WriteString(s[parsedTo:ids[0]])
		reqID := s[ids[0]:ids[1]]
		parsedTo = ids[1]
		// As per REQ-TRAQ-SWH-002:
		// REQ-[project/system number]-[project/system abbreviation]-[SSS or SWH or SWL or HWH or HWL]-[a unique alphanumeric sequence],
		project := s[ids[2]:ids[3]]
		reqType := s[ids[4]:ids[5]]
		if len(ids) != 8 {
			// This should not happen.
			return "", fmt.Errorf("regexp cannot be used, please file a bug in Devtools: %q", ids)
		}
		docType, ok := config.ReqTypeToDocType[reqType]
		if !ok {
			return "", fmt.Errorf("unknown requirement type: %q (in %q)", reqType, reqID)
		}
		docID, ok := config.DocTypeToDocId[docType]
		if !ok {
			return "", fmt.Errorf("doc type has no doc id associated: %q (in %q)", docType, reqID)
		}
		// For example: TRAQ-100-ORD
		name := fmt.Sprintf("%s-%s-%s", project, docID, docType)
		url := fmt.Sprintf("http://a.daedalean.ai/docs/%s/%s/%s.pdf#%s", repo, dirInRepo, name, reqID)
		res.WriteString(fmt.Sprintf(`
\begin_inset CommandInset href
LatexCommand href
name "%s"
target "%s"

\end_inset

`, reqID, url))
	}
	res.WriteString(s[parsedTo:len(s)])
	return res.String(), nil
}
