/*
Wrapper functions the golang command interface.
*/

package linepipes

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// global flag controlling debug output
var Verbose = false

// @llr REQ-TRAQ-SWL-48
func Run(prog string, args ...string) (lines chan string, errs chan error) {
	return RunWithInput(prog, os.Stdin, args...)
}

// @llr REQ-TRAQ-SWL-48
func RunWithInput(prog string, input io.Reader, args ...string) (lines chan string, errs chan error) {
	lines = make(chan string)
	errs = make(chan error, 1)
	escapedCommand := EscapeCommand(prog, args...)
	if Verbose {
		log.Println("Executing:", escapedCommand)
	}
	cmd := exec.Command(prog, args...)
	cmd.Stdin = input
	pipeReader, pipeWriter, err := os.Pipe()
	if err != nil {
		errs <- err
		close(lines)
		close(errs)
		return lines, errs
	}
	cmd.Stdout = pipeWriter
	cmd.Stderr = pipeWriter
	if err := cmd.Start(); err != nil {
		errs <- err
		close(lines)
		close(errs)
		return lines, errs
	}
	go func() {
		defer close(lines)
		s := bufio.NewScanner(pipeReader)
		for s.Scan() {
			lines <- s.Text()
		}
	}()
	go func() {
		defer close(errs)
		if err := cmd.Wait(); err != nil {
			errs <- fmt.Errorf("command failed: %s: %s", err, escapedCommand)
		}
		pipeWriter.Close()
	}()
	return lines, errs
}

// @llr REQ-TRAQ-SWL-48
func Single(lines <-chan string, errors <-chan error) (string, error) {
	var s string
	var count int
	for line := range lines {
		s = line
		count += 1
	}
	if err, _ := <-errors; err != nil {
		return s, err
	}
	if count != 1 {
		return s, fmt.Errorf("Expected a single line, got %d", count)
	}
	return s, nil
}

// @llr REQ-TRAQ-SWL-48
func All(lines <-chan string, errors <-chan error) (string, error) {
	var buffer bytes.Buffer
	for line := range lines {
		buffer.WriteString(line)
		buffer.WriteByte('\n')
	}
	if err, _ := <-errors; err != nil {
		return "", err
	}
	return buffer.String(), nil
}

// unsafeCharRe contains the list of safe shell chars from Python's shlex.quote implementation.
var unsafeCharRe = regexp.MustCompile(`[^\w@%+=:,./-]`)

// @llr REQ-TRAQ-SWL-48
func EscapeArg(arg string) string {
	if unsafeCharRe.MatchString(arg) {
		return fmt.Sprintf("'%s'", strings.Replace(arg, `'`, `'"'"'`, -1))
	}
	return arg
}

// EscapeCommand formats the command such that it can be copy/pasted to be run.
// @llr REQ-TRAQ-SWL-48
func EscapeCommand(prog string, args ...string) string {
	res := make([]string, 0, len(args)+1)
	res = append(res, EscapeArg(prog))
	for _, arg := range args {
		res = append(res, EscapeArg(arg))
	}
	return strings.Join(res, " ")
}
