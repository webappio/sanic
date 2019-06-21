package util

import (
	"fmt"
	"os"
	"strings"
)

//ExpandUser turns, e.g., ~/hello to /home/youruser/hello
//it does *not* work on ~user/blah paths, however
func ExpandUser(s string) (string, error)  {
	homedir := os.Getenv("HOME")
	if homedir == "" {
		return "", fmt.Errorf("Tried to expand %s, but there was no HOME set.", s)
	}
	if strings.HasPrefix(s, "~/") {
		return homedir+"/"+s[2:], nil
	}
	if s[0] == '~' {
		return "", fmt.Errorf("Unrecognized path type: %s, did not expect it to start with ~, s")
	}
	return s, nil
}