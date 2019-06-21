package localdev

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"sigs.k8s.io/kind/pkg/container/cri"
	"strings"
)

func (provisioner *ProvisionerLocalDev) liveMounts() []cri.Mount {
	var liveMounts []cri.Mount

	usr, err := user.Current()
	if err == nil {
		hostHome, err := filepath.EvalSymlinks(usr.HomeDir)
		if err != nil {
			panic(err) //these errors are all catastrophic, like symlink loops in home folder!
		}
		liveMounts = append(liveMounts, cri.Mount{
			ContainerPath: "/hosthome",
			HostPath:      hostHome,
			Readonly:      true,
		})
	}

	if _, err := os.Stat("/mnt"); err == nil {
		liveMounts = append(liveMounts, cri.Mount{
			ContainerPath: "/mnt",
			HostPath:      "/mnt",
			Readonly:      true,
		})
	}
	return liveMounts
}

//InClusterDir returns a path relative to the mounts in the liveMounts() function above.
//For example, /home/(your username)/projects/sanic -> /hosthome/projects/sanic
func (provisioner *ProvisionerLocalDev) InClusterDir(hostDir string) string {
	hostDir, err := filepath.EvalSymlinks(hostDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not evaluate symlinks in path %s: %s", hostDir, err.Error())
		return "<INVALID_MOUNTPOINT_SYMLINKS>"
	}

	for _, mnt := range provisioner.liveMounts() {
		if strings.HasPrefix(hostDir, mnt.HostPath) {
			relDir, err := filepath.Rel(mnt.HostPath, hostDir)
			if err != nil {
				panic(err) //shouldn't happen: mounts are absolute
			}
			return mnt.ContainerPath + "/" + relDir
		}
	}

	var liveMountedFolders []string
	for _, mnt := range provisioner.liveMounts() {
		liveMountedFolders = append(liveMountedFolders, mnt.HostPath)
	}
	fmt.Fprintf(
		os.Stderr,
		"Warning: Directory %s is not in a live-mounted folder (%s): Live mounting it will not work.\n",
		hostDir,
		strings.Join(liveMountedFolders, ", "),
	)
	return "dir_is_not_in_mounted_dir"
}
