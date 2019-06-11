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

func projectDirEnvVar(projectRoot string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not find your home directory, live-mounting will not work: %s\n", err.Error())
		return "PROJECT_DIR=<error: no home folder>"
	}

	homeDir, err = filepath.EvalSymlinks(homeDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not evaluate symlinks in your home directory, live-mounting will not work: %s\n", err.Error())
		return "PROJET_DIR=<error: home symlink resolution>"
	}

	projectRoot, err = filepath.EvalSymlinks(projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not evaluate symlinks on the path to the project root, live-mounting will not work: %s\n", err.Error())
		return "PROJET_DIR=<error: project root symlink resolution>"
	}

	if strings.HasPrefix(projectRoot, homeDir) {
		sourceDirRelHome, err := filepath.Rel(homeDir, projectRoot)
		if err != nil {
			//shouldn't happen: homeDir is absolute
			panic(err)
		}
		return "PROJECT_DIR=/hosthome/"+sourceDirRelHome
	}

	fmt.Fprintf(os.Stderr, "Warning: Your project is in %s, which is not in your home folder %s: Live mounting will not work.\n", shl.GetSanicRoot(), homeDir)
	return "PROJECT_DIR=project_source_is_not_in_home"
}

func runTemplater(folderIn, folderOut, templaterImage string) error {
	shl, err := shell.Current()
	if err != nil {
		return err
	}

	provisioner, err := provisioners.GetProvisioner()
	if err != nil {
		return err
	}
	registry, err := provisioner.Registry()
	if err != nil {
		return err
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
		"-e", projectDirEnvVar(shl.GetSanicRoot()),
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
	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	cmd.Stdout = stdout
	err := cmd.Run()
	if err == nil {
		fmt.Print(stdout.String())
	} else {
		fmt.Fprint(os.Stderr, stderr.String())
	}
	return err
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

	provisioner, err := provisioners.GetProvisioner()
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
