package provisioners

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	kubeclientcmd "k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/kind/pkg/cluster"
	kindconfig "sigs.k8s.io/kind/pkg/cluster/config"
	"sigs.k8s.io/kind/pkg/cluster/config/encoding"
	"sigs.k8s.io/kind/pkg/cluster/create"
	"time"
)

//ProvisionerLocalDev is a provisioner which uses "kind" to set up a local, 4-node development kubernetes cluster
//within docker itself.
type ProvisionerLocalDev struct{}

var kindContext = cluster.NewContext("sanic")

func kubeNodeReady(node corev1.Node) bool {
	ready := false

	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			ready = condition.Status == corev1.ConditionTrue
		}
	}

	return ready
}

func checkCluster(dockerCli *client.Client, kube *kubernetes.Clientset) error {
	clusterContainers, err := dockerCli.ContainerList(
		context.Background(),
		types.ContainerListOptions{
			All:     true,
			Filters: filters.NewArgs(filters.Arg("label", "io.k8s.sigs.kind.cluster")),
		})
	if err != nil {
		return err
	}

	requiredContainersRunning := map[string]bool{
		"/sanic-worker":        false,
		"/sanic-worker2":       false,
		"/sanic-worker3":       false,
		"/sanic-control-plane": false,
	}

	var nodeContainerIDs []string

	for _, container := range clusterContainers {
		if _, ok := requiredContainersRunning[container.Names[0]]; ok {
			requiredContainersRunning[container.Names[0]] = container.State == "running"
			nodeContainerIDs = append(nodeContainerIDs, container.ID)
		}
	}
	for containerName, status := range requiredContainersRunning {
		if !status {
			return fmt.Errorf("at least one required container isn't running: %s", containerName)
		}
	}
	nodes, err := kube.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("could not list kubernetes nodes: %s", err.Error())
	}
	if len(nodes.Items) != len(requiredContainersRunning) {
		return fmt.Errorf("some nodes have been removed/crashed. only %d/%d were running",
			len(nodes.Items), len(requiredContainersRunning))
	}
	for _, node := range nodes.Items {
		if !kubeNodeReady(node) {
			return fmt.Errorf("a node was not ready.\nTo note: after deploying initially, " +
				"wait at least 30 seconds before deploying again to let the cluster start fully")
		}
	}
	return nil
}

func deleteClusterContainers(dockerCli *client.Client) error {
	clusterContainers, err := dockerCli.ContainerList(
		context.Background(),
		types.ContainerListOptions{
			All:     true,
			Filters: filters.NewArgs(filters.Arg("label", "io.k8s.sigs.kind.cluster")),
		})
	if err != nil {
		return err
	}
	for _, container := range clusterContainers {
		if err = dockerCli.ContainerRemove(context.Background(), container.ID, types.ContainerRemoveOptions{Force: true}); err != nil {
			return err
		}
	}
	return nil
}

func (provisioner *ProvisionerLocalDev) KubeConfigLocation() string {
	return kindContext.KubeConfigPath()
}

func (provisioner *ProvisionerLocalDev) EnsureCluster() error {
	dockerCli, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.24"))
	if err != nil {
		return fmt.Errorf("could not connect to docker successfully. version 1.12.1 or higher is required.\n, %s", err.Error())
	}
	kubeConfig, err := kubeclientcmd.BuildConfigFromFlags("", provisioner.KubeConfigLocation())
	var clusterError error
	if err != nil {
		clusterError = fmt.Errorf("kind config did not exist, cluster has not been initialized")
	} else {
		//kind's kubernetes config exists
		kube, err := kubernetes.NewForConfig(kubeConfig)
		if err != nil {
			clusterError = fmt.Errorf("could not connect to kubernetes in kind, it is likely not running: %s", err.Error())
		} else {
			clusterError = checkCluster(dockerCli, kube)
		}
	}

	if clusterError == nil {
		return nil //nothing to do, cluster is healthy
	} else {
		fmt.Printf("Creating a new cluster, old one cannot be used: %s\n", clusterError.Error())
		fmt.Println("This takes between 1 and 10 minutes, depending on your internet connection speed.")
	}
	cfg := kindconfig.Cluster{}
	encoding.Scheme.Default(&cfg)
	cfg.Nodes = []kindconfig.Node{
		{
			Role: kindconfig.ControlPlaneRole,
		},
		{
			Role: kindconfig.WorkerRole,
		},
		{
			Role: kindconfig.WorkerRole,
		},
		{
			Role: kindconfig.WorkerRole,
		},
	}

	//TODO HACK: kind does not always work if the containers are not manually removed first
	if err := deleteClusterContainers(dockerCli); err != nil {
		return fmt.Errorf("could not delete existing containers to run cluster setup: %s", err.Error())
	}

	return kindContext.Create(&cfg, create.Retain(false), create.WaitForReady(time.Duration(0)))
}
