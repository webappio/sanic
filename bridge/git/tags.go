package git

import (
	"bytes"
	"fmt"
	"io/ioutil"
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
	if err == nil {
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

//GetGitRoot returns the directory which contains the .git folder
func GetGitRoot(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Dir = dir
	cmd.Stdout = stdout
	err := cmd.Run()
	if err != nil {
		fmt.Fprint(os.Stderr, stderr.String())
		return "", fmt.Errorf("could not get the git directory. Is git installed and setup? %s", err.Error())
	}
	return strings.TrimSpace(stdout.String()), nil
}

//GetCurrentTreeHash returns a hash of the git repository, consisting of the currently
//commited files, as well as any unstaged changes in the provided directories
func GetCurrentTreeHash(rootDir string, unstagedFiles ...string) (string, error) {
	rootDir, err := GetGitRoot(rootDir)
	if err != nil {
		return "", err
	}

	tmpindex, err := ioutil.TempFile("", "sanicindex")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpindex.Name())

	//See http://web.archive.org/web/20190606210911/https://stackoverflow.com/questions/23816330/compute-git-hash-of-all-uncommitted-code
	currIndexData, err := ioutil.ReadFile(rootDir + "/.git/index")
	if err != nil {
		return "", err
	}
	_, err = tmpindex.Write(currIndexData)
	if err != nil {
		return "", err
	}

	stderr := &bytes.Buffer{}

	cmd := exec.Command("git", append([]string{"add", "-A"}, unstagedFiles...)...)
	cmd.Dir = rootDir
	cmd.Env = append(os.Environ(), "GIT_INDEX_FILE="+tmpindex.Name())
	cmd.Stderr = stderr
	err = cmd.Run()
	if err != nil {
		fmt.Fprint(os.Stderr, stderr.String())
		return "", err
	}

	stdout := &bytes.Buffer{}
	stderr = &bytes.Buffer{}

	cmd = exec.Command("git", "write-tree")
	cmd.Dir = rootDir
	cmd.Env = append(os.Environ(), "GIT_INDEX_FILE="+tmpindex.Name())
	cmd.Stderr = stderr
	cmd.Stdout = stdout
	err = cmd.Run()
	if err != nil {
		fmt.Fprint(os.Stderr, stderr.String())
		return "", err
	}
	return strings.TrimSpace(stdout.String())[:12], nil

}
