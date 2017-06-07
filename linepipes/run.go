package linepipes

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// global flag controlling debug output
var Verbose = false

func Run(prog string, args ...string) (lines chan string, errors chan error) {
	lines = make(chan string)
	errors = make(chan error, 1)
	if Verbose {
		log.Println("Executing:", prog, strings.Join(args, " "))
	}
	cmd := exec.Command(prog, args...)
	cmd.Stdin = os.Stdin
	pipeReader, pipeWriter, err := os.Pipe()
	if err != nil {
		errors <- err
		close(lines)
		close(errors)
		return lines, errors
	}
	cmd.Stdout = pipeWriter
	cmd.Stderr = pipeWriter
	if err := cmd.Start(); err != nil {
		errors <- err
		close(lines)
		close(errors)
		return lines, errors
	}
	go func() {
		defer close(lines)
		s := bufio.NewScanner(pipeReader)
		for s.Scan() {
			lines <- s.Text()
		}
	}()
	go func() {
		defer close(errors)
		if err := cmd.Wait(); err != nil {
			errors <- err
		}
		pipeWriter.Close()
	}()
	return lines, errors
}

func writeLine(file *os.File, line string) error {
	if _, err := file.WriteString(line); err != nil {
		return err
	}
	if _, err := file.Write([]byte{'\n'}); err != nil {
		return err
	}
	return nil
}

func Out(lines <-chan string, errors <-chan error) error {
	if err := Redirect(lines, os.Stdout); err != nil {
		return err
	}
	if err, _ := <-errors; err != nil {
		return err
	}
	return nil
}

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

func Last(lines <-chan string, errors <-chan error) (string, error) {
	var s string
	for v := range lines {
		if v != "" {
			s = v
		}
	}
	if err, _ := <-errors; err != nil {
		return s, err
	}
	return s, nil
}

func Redirect(lines <-chan string, file *os.File) error {
	for line := range lines {
		if err := writeLine(file, line); err != nil {
			return err
		}
	}
	return nil
}

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
