package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

//GetCurrentTag returns the tag of the HEAD commit of the git repository in the specified directory
//It returns an error if git isn't installed, or if there is some sort of I/O problem
//It does not return an error if git exits with a non-zero exit code, it assumes this means there are no tags
func GetCurrentTag(dir string) (string, error) {
	cmd := exec.Command("git", "describe", "--tags")
	stdout := &bytes.Buffer{}
	cmd.Dir = dir
	cmd.Stdout = stdout
	err := cmd.Start()
	if err != nil {
		return "", fmt.Errorf("could not get tags, is git installed? %s", err.Error())
	}
	err = cmd.Wait()
	if err == nil  {
		tagsString := strings.TrimSpace(stdout.String())
		spaceIdx := strings.Index(tagsString, " ")
		if spaceIdx != -1 {
			tagsString = tagsString[:spaceIdx]
		}
		return tagsString, nil
	}
	if _, ok := err.(*exec.ExitError); ok {
		return "", nil //no tags found
	}
	return "", err
}

//GetCurrentShortHash returns the short hash (i.e., 7 char prefix of current commit) of the git repository
// in the specified directory. It returns an error if the hash cannot be calculated for any reason
func GetCurrentShortHash(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Dir = dir
	cmd.Stdout = stdout
	err := cmd.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, stderr.String())
		return "", fmt.Errorf("could not get the commit hash. Is git installed, and have you committed? %s", err.Error())
	}
	return strings.TrimSpace(stdout.String()), nil
}