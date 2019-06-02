package provisioners

import (
	"bytes"
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
	"io"
	"os"
	"os/exec"
	"sigs.k8s.io/kind/pkg/cluster"
	kindconfig "sigs.k8s.io/kind/pkg/cluster/config"
	"sigs.k8s.io/kind/pkg/cluster/config/encoding"
	"sigs.k8s.io/kind/pkg/cluster/create"
	kindnode "sigs.k8s.io/kind/pkg/cluster/nodes"
	"strings"
	"time"
)

//ProvisionerLocalDev is a provisioner which uses "kind" to set up a local, 4-node development kubernetes cluster
//within docker itself.
type ProvisionerLocalDev struct{}

var kindContext = cluster.NewContext("sanic")

func clusterNodes() ([]kindnode.Node, error) {
	return kindnode.List("label=io.k8s.sigs.kind.cluster=sanic")
}

func (provisioner *ProvisionerLocalDev) checkClusterReady() error {
	cmd := exec.Command(
		"kubectl",
		"--kubeconfig="+provisioner.KubeConfigLocation(),
		"get",
		"nodes",
		"-o",
		"jsonpath={.items..status.conditions[-1:].lastTransitionTime}\t{.items..status.conditions[-1:].status}",
	)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("could not check if the cluster was running: %s %s", err.Error(), stderr.String())
	}
	//output is, e.g., "2019-06-02T01:04:02Z 2019-06-02T01:04:18Z 2019-06-02T01:04:17Z 2019-06-02T01:04:17Z\tTrue True True True"
	split := strings.Split(stdout.String(), "\t")
	if len(split) != 2 {
		return fmt.Errorf("got invalid kubernetes output while checking if cluster was running: \"%s\"", stdout.String())
	}
	nodeTimes := strings.Split(split[0], " ")
	nodesReady := strings.Split(split[1], " ")

	if len(nodeTimes) != 4 {
		return fmt.Errorf("some nodes were not running, we were expecting 4 (3 workers + one master node)")
	}

	allNodesReady := true
	for _, nodeReady := range nodesReady {
		nodeReady = strings.TrimSpace(nodeReady)
		if nodeReady != "True" {
			allNodesReady = false
		}
	}
	if allNodesReady {
		return nil
	}

	statusChangeRecent := false

	for _, timeString := range nodeTimes {
		timeString = strings.TrimSpace(timeString)
		//TODO will kubernetes ever give iso 8601 formatted date which is not this specific format?
		//Mon Jan 2 15:04:05 MST 2006
		statusTime, err := time.Parse("2006-01-02T15:04:05Z", timeString)
		if err != nil {
			return fmt.Errorf("got invalid kubernetes output while checking how long cluster has been running: %s", timeString)
		}
		if statusTime.Add(time.Minute).After(time.Now()) {
			//status has changed less than a minute ago
			statusChangeRecent = true
		}
	}

	if statusChangeRecent {
		for {
			fmt.Print("do you want to redeploy recently started kubernetes cluster? [y/N]: ")
			var resp string
			fmt.Scanln(&resp)
			switch resp{
			case "y", "Y":
				return fmt.Errorf("some nodes weren't ready, and you chose to redeploy")
			case "n", "N", "":
				return nil
			default:
				fmt.Printf("Did not understand response: %s, expected y/n\n", resp)
			}
		}
	}
	return fmt.Errorf("cluster is not ready, and has not been for over a minute")
}

func (provisioner *ProvisionerLocalDev) checkCluster() error {
	nodes, err := clusterNodes()
	if err != nil {
		return err
	}

	requiredContainersRunning := map[string]*kindnode.Node{
		"sanic-worker":        nil,
		"sanic-worker2":       nil,
		"sanic-worker3":       nil,
		"sanic-control-plane": nil,
	}

	for _, node := range nodes {
		if _, ok := requiredContainersRunning[node.Name()]; ok {
			requiredContainersRunning[node.Name()] = &node
		}
	}

	if len(nodes) == 0 {
		return fmt.Errorf("no nodes were running, cluster has to be provisioned once per docker engine restart")
	}

	if len(nodes) != len(requiredContainersRunning) {
		return fmt.Errorf("some nodes have been removed/crashed. only %d/%d were running",
			len(nodes), len(requiredContainersRunning))
	}
	for _, node := range requiredContainersRunning {
		if node == nil {
			return fmt.Errorf("some nodes were not running while others were, try deleting your cluster containers with docker rm")
		}
	}

	return provisioner.checkClusterReady()
}

func deleteClusterContainers() error {
	nodes, err := clusterNodes()
	if err != nil {
		return err
	}
	eg := errgroup.Group{}
	for _, node := range nodes {
		name := node.Name()
		eg.Go(func() error {
			cmd := exec.Command("docker", "rm", "-f", name)
			return cmd.Run()
		})
	}
	return eg.Wait()
}

//KubeConfigLocation returns the path to the kubectl configuration for this provisioner
func (provisioner *ProvisionerLocalDev) KubeConfigLocation() string {
	return kindContext.KubeConfigPath()
}

const traefikIngressYaml = `
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: traefik-ingress-controller
rules:
  - apiGroups:
      - ""
    resources:
      - services
      - endpoints
      - secrets
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - extensions
    resources:
      - ingresses
    verbs:
      - get
      - list
      - watch

---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: traefik-ingress-controller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: traefik-ingress-controller
subjects:
- kind: ServiceAccount
  name: traefik-ingress-controller
  namespace: kube-system

---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: traefik-ingress-controller
  namespace: kube-system

---
kind: Deployment
apiVersion: extensions/v1beta1
metadata:
  name: traefik-ingress-controller
  namespace: kube-system
  labels:
    k8s-app: traefik-ingress-lb
spec:
  replicas: 1
  selector:
    matchLabels:
      k8s-app: traefik-ingress-lb
  template:
    metadata:
      labels:
        k8s-app: traefik-ingress-lb
        name: traefik-ingress-lb
    spec:
      serviceAccountName: traefik-ingress-controller
      terminationGracePeriodSeconds: 60
      hostNetwork: true
      containers:
      - image: traefik
        name: traefik-ingress-lb
        ports:
        - name: http
          containerPort: 80
        - name: admin
          containerPort: 8080
        args:
        - --api
        - --kubernetes
        - --logLevel=INFO
`

func (provisioner *ProvisionerLocalDev) startIngressController() error {
	cmd := exec.Command("kubectl", "apply", "-f", "-")
	cmd.Env = append(os.Environ(), "KUBECONFIG="+provisioner.KubeConfigLocation())
	cmd.Stdin = bytes.NewBufferString(traefikIngressYaml)
	errBuffer := &bytes.Buffer{}
	cmd.Stdout = os.Stdout
	cmd.Stderr = errBuffer
	err := cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	if err != nil {
		fmt.Fprint(os.Stderr, errBuffer.String())
	}
	return err
}

//EnsureCluster for localdev is a wrapper around "kind", which sets up a 4-node kubernetes cluster in docker itself.
func (provisioner *ProvisionerLocalDev) EnsureCluster() error {
	clusterError := provisioner.checkCluster()

	if clusterError == nil {
		return nil //nothing to do, cluster is healthy
	}
	fmt.Printf("Creating a new cluster, old one cannot be used: %s\n", clusterError.Error())
	fmt.Println("This takes between 1 and 10 minutes, depending on your internet connection speed.")
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
	if err := deleteClusterContainers(); err != nil {
		return fmt.Errorf("could not delete existing containers to run cluster setup: %s", err.Error())
	}

	err := kindContext.Create(&cfg, create.Retain(false))
	if err != nil {
		return err
	}
	err = provisioner.startIngressController()
	if err != nil {
		return fmt.Errorf("could not start the ingress controller: %s", err.Error())
	}
	//TODO message about where the webserver is available
	return err
	return nil
}


