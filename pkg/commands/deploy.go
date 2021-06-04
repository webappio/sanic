package commands

import (
	"bytes"
	"fmt"
	"github.com/layer-devops/sanic/pkg/bridge/git"
	"github.com/layer-devops/sanic/pkg/config"
	"github.com/layer-devops/sanic/pkg/provisioners/provisioner"
	"github.com/layer-devops/sanic/pkg/shell"
	"github.com/layer-devops/sanic/pkg/util"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"text/template"
)

func getenv(key string, default_ ...string) string {
	env := os.Getenv(key)
	if env != "" {
		return env
	}
	return strings.Join(default_, " ")
}

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
	fmt.Println("Pulling the templater image " + image + "...")
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
	log.Printf("hey!!! I'm in!!!\n")
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
		templaterImage = templaterImage + ":latest"
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
	)
	
	for _, env := range os.Environ() {
		cmd.Args = append(cmd.Args, "-e", env)	
	}
	
	cmd.Args = append(cmd.Args, 
		"-e", "SANIC_ENV="+shl.GetSanicEnvironment(),
		"-e", "REGISTRY_HOST="+registry,
		"-e", "IMAGE_TAG="+buildTag,
		"-e", "PROJECT_DIR="+provisioner.InClusterDir(shl.GetSanicRoot()),
		"-e", "NAMESPACE="+namespace,
		templaterImage)
	
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
	files, err := filepath.Glob(folderIn+"/*.tmpl")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not find the template files at /in, expecting, e.g., /in/a.tmpl, /in/b.tmpl to exist. Error: %s\n", err.Error())
		syscall.Exit(1)
	}
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "No configuration files were specified at /in/... with suffix '.tmpl'\n")
		syscall.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not delete the contents of the output directory. Is it mounted read/write? %s\n", err.Error())
		syscall.Exit(1)
	}

	fmt.Printf("Templating %d config files...\n", len(files))

	os.Setenv("SANIC_ENV",shl.GetSanicEnvironment())
	os.Setenv("REGISTRY_HOST",registry)
	os.Setenv("IMAGE_TAG", buildTag)
	os.Setenv("PROJECT_DIR", provisioner.InClusterDir(shl.GetSanicRoot()))
	os.Setenv("NAMESPACE", namespace)

	log.Printf("!!!project dir is %v\n", os.Getenv("PROJECT_DIR"))

	for _, templatepath := range files {
		templateName := strings.TrimSuffix(filepath.Base(templatepath), ".tmpl")
		fmt.Printf("Running template %s...\n", templateName)
		t, err := template.New(
			filepath.Base(templatepath),
		).Funcs(
			map[string]interface{}{
				"getenv": getenv,
			},
		).ParseFiles(
			append([]string{templatepath}, files...)...
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not compile the templates at /in/...: %s\n", err.Error())
			syscall.Exit(1)
		}

		//400: not writable by user intentionally, these files are auto generated
		outFile, err := os.OpenFile(folderOut+"/"+templateName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not open the file at %s for writing. Did you run this image with -v (output path on host):/out ?\n", folderOut+"/"+templateName)
			syscall.Exit(1)
		}
		outFile.WriteString("#WARNING: THIS FILE IS AUTOMATICALLY GENERATED, DO NOT EDIT IT DIRECTLY OR COMMIT IT\n")

		err = t.Execute(outFile, nil)
		outFile.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not execute the template at %s: %s\n", templatepath, err.Error())
			syscall.Exit(1)
		}
		err = os.Chown(folderOut+"/"+templateName, 1001, 1001)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not chown the template %s to %d/%d", templateName, 1001, 1001)
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
		return cli.NewExitError(fmt.Sprintf("The input folder at %s could not be read. Does it exist? %s\nSee https://github.com/layer-devops/sanic-site for an example.", folderIn, err.Error()), 1)
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
	log.Printf("%v!!!\n", cfg.Deploy.TemplaterImage)
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
		return cli.NewExitError(fmt.Sprintf("could not apply templates in!!! %s: %s", folderOut, err.Error()), 1)
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
