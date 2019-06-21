package commands

import (
	"bytes"
	"fmt"
	"github.com/distributed-containers-inc/sanic/bridge/git"
	"github.com/distributed-containers-inc/sanic/config"
	"github.com/distributed-containers-inc/sanic/provisioners"
	"github.com/distributed-containers-inc/sanic/shell"
	"github.com/distributed-containers-inc/sanic/util"
	"github.com/urfave/cli"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func clearYamlsFromDir(folderOut string) error {
	files, err := filepath.Glob(folderOut + "/*.yaml")
	if err != nil {
		return err
	}
	for _, f := range files {
		err = os.Remove(f)
		if err != nil {
			return err
		}
	}
	return nil
}

func runTemplater(folderIn, folderOut, templaterImage string) error {
	shl, err := shell.Current()
	if err != nil {
		return err
	}

	provisioner, err := getProvisioner()
	if err != nil {
		return err
	}

	registry, err := provisioner.Registry()
	if err != nil {
		return err
	}
	if strings.HasPrefix(registry, "http://") {
		registry = registry[len("http://"):]
	} else if strings.HasPrefix(registry, "https://") {
		registry = registry[len("https://"):]
	} else {
		panic(fmt.Errorf("Got an invalid value for registry, expected it to start with http or https: %s", registry))
	}

	services, err := util.FindServices()
	if err != nil {
		return err
	}

	buildTag, err := git.GetCurrentTreeHash(shl.GetSanicRoot(), services...)
	if err != nil {
		return err
	}

	err = clearYamlsFromDir(folderOut)
	if err != nil {
		return err
	}

	tempFolderOut, err := ioutil.TempDir("", "sanicdeploy")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempFolderOut)

	cmd := exec.Command(
		"docker",
		"run",
		"--rm",
		"-v", folderIn+"/:/in:ro",
		"-v", tempFolderOut+"/:/out",
		"-e", "SANIC_ENV="+shl.GetSanicEnvironment(),
		"-e", "REGISTRY_HOST="+registry,
		"-e", "IMAGE_TAG="+buildTag,
		"-e", "PROJECT_DIR="+provisioner.InClusterDir(shl.GetSanicRoot()),
		templaterImage,
	)
	stderrBuffer := &bytes.Buffer{}
	cmd.Stderr = stderrBuffer
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf(
			"could not generate the kubernetes configurations from %s: %s\n%s",
			folderIn, err.Error(), stderrBuffer.String())
	}
	templatedFiles, err := filepath.Glob(tempFolderOut + "/*")
	if err != nil {
		return fmt.Errorf("could not read the templated deployment files: %s", err.Error())
	}
	for _, f := range templatedFiles { //we volume mount a temp folder and copy from there to ensure files are owned by the user and not root
		fileData, err := ioutil.ReadFile(f)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(folderOut+"/"+filepath.Base(f), fileData, 0400)
		if err != nil {
			return fmt.Errorf("could not write the template %s to the directory %s: %s", f, folderOut, err.Error())
		}
	}
	return nil
}

func kubectlApplyFolder(folder string, provisioner provisioners.Provisioner) error {
	//TODO NOT PRODUCTION READY: --prune might be destructive
	cmd := exec.Command("kubectl", "--kubeconfig", provisioner.KubeConfigLocation(), "apply", "-f", folder, "--prune", "--all")
	out := &bytes.Buffer{}
	cmd.Stdout = out
	cmd.Stderr = out
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf(out.String())
	}
	return nil
}

func deployCommandAction(cliContext *cli.Context) error {
	cfg, err := config.Read()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	shl, err := shell.Current()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	folderIn, err := filepath.Abs(shl.GetSanicRoot() + "/" + cfg.Deploy.Folder + "/in")
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	folderOut, err := filepath.Abs(shl.GetSanicRoot() + "/" + cfg.Deploy.Folder + "/out")
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	provisioner, err := getProvisioner()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	err = provisioner.EnsureCluster()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	err = runTemplater(folderIn, folderOut, cfg.Deploy.TemplaterImage)
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("could not compile templates: %s", err.Error()), 1)
	}
	err = kubectlApplyFolder(folderOut, provisioner)
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("could not apply templates in %s: %s", folderOut, err.Error()), 1)
	}
	edgeNodes, err := provisioner.EdgeNodes()
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("could not find edge routers: %s", err.Error()), 1)
	}
	if len(edgeNodes) == 0 {
		//this shouldn't happen: environment is misconfigured?
		return cli.NewExitError("there are no edge routers in this environment. Try reprovisioning your cluster", 1)
	}
	fmt.Printf("Configured HTTP services are available at http://%s\n", edgeNodes[rand.Intn(len(edgeNodes))])
	return nil
}

var deployCommand = cli.Command{
	Name:   "deploy",
	Usage:  "deploy some (or all, by default) services",
	Action: deployCommandAction,
}
