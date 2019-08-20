package commands

import (
	"bytes"
	"fmt"
	"github.com/distributed-containers-inc/sanic/bridge/git"
	"github.com/distributed-containers-inc/sanic/config"
	"github.com/distributed-containers-inc/sanic/provisioners/provisioner"
	"github.com/distributed-containers-inc/sanic/shell"
	"github.com/distributed-containers-inc/sanic/util"
	"github.com/pkg/errors"
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

func pullImageIfNotExists(image string) error {
	cmd := exec.Command("docker", "inspect", image)
	if cmd.Run() == nil {
		return nil //already exists
	}
	fmt.Println("Pulling the templater image "+image+"...")
	cmd = exec.Command(
		"docker",
		"pull",
		image,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runTemplater(folderIn, folderOut, templaterImage, namespace string) error {
	if namespace == "" {
		namespace = "<ERROR_NAMESPACE_NOT_DEFINED_IN_THIS_ENV>"
	}

	cfg, err := config.Read()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	shl, err := shell.Current()
	if err != nil {
		return err
	}
	provisioner, err := getProvisioner()
	if err != nil {
		return err
	}
	registry, _, err := provisioner.Registry()
	if err != nil {
		return err
	}
	services, err := util.FindServices(shl.GetSanicRoot(), cfg.Build.IgnoreDirs)
	if err != nil {
		return err
	}
	var serviceDirectories []string
	for _, service := range services {
		serviceDirectories = append(serviceDirectories, service.Dir)
	}
	buildTag, err := git.GetCurrentTreeHash(shl.GetSanicRoot(), serviceDirectories...)
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

	if !strings.Contains(templaterImage, ":") {
		templaterImage = templaterImage+":latest"
	}

	err = pullImageIfNotExists(templaterImage)
	if err != nil {
		return fmt.Errorf("could not pull the templater image %s: %s", templaterImage, err)
	}

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
		"-e", "NAMESPACE="+namespace,
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

func createNamespace(namespace string, provisioner provisioner.Provisioner) error {
	cmd, err := provisioner.KubectlCommand("create", "namespace", namespace)
	if err != nil {
		return errors.Wrapf(err, "error creating namespace %s", namespace)
	}
	out := &bytes.Buffer{}
	cmd.Stdout = out
	cmd.Stderr = out
	err = cmd.Run()
	if err != nil && !strings.Contains(out.String(), "AlreadyExists") {
		return errors.New(strings.TrimSpace(out.String()))
	}
	return nil
}

func kubectlApplyFolder(folder string, provisioner provisioner.Provisioner) error {
	cmd, err := provisioner.KubectlCommand("apply", "-f", folder)
	if err != nil {
		return errors.Wrapf(err, "error while applying folder %s", folder)
	}
	out := &bytes.Buffer{}
	cmd.Stdout = out
	cmd.Stderr = out
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf(strings.TrimSpace(out.String()))
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
	env, err := cfg.CurrentEnvironment(shl)
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

	if _, err := os.Stat(folderIn); err != nil {
		return cli.NewExitError(fmt.Sprintf("The input folder at %s could not be read. Does it exist? %s\nSee https://github.com/distributed-containers-inc/sanic-site for an example.", folderIn, err.Error()), 1)
	}
	err = os.MkdirAll(folderOut, 0750)
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("The deployment output folder at %s could not be created: %s", folderOut, err.Error()), 1)
	}

	provisioner, err := getProvisioner()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	err = provisioner.EnsureCluster()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	err = runTemplater(folderIn, folderOut, cfg.Deploy.TemplaterImage, env.Namespace)
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("could not compile templates: %s", err.Error()), 1)
	}
	if env.Namespace != "" {
		err = createNamespace(env.Namespace, provisioner)
		if err != nil {
			return cli.NewExitError(fmt.Sprintf(
				"namespace %s defined in sanic.yaml for this environment couldn't be created: %s",
				env.Namespace, err.Error(),
			), 1)
		}
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
