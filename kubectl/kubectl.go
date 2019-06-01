package kubectl

import (
	"fmt"
	"os/exec"
)

//GetKubectlExecutablePath returns the absolute path to the kubectl executable (e.g., from PATH)
func GetKubectlExecutablePath() (string, error) {
	kubeExecutableLocation, err := exec.LookPath("kubectl")
	if err != nil {
		return "", fmt.Errorf(
			"the kubectl executable was not found on your PATH, it needs to be installed manually. Error:%s\n",
			err.Error())
	}
	return kubeExecutableLocation, nil
}
