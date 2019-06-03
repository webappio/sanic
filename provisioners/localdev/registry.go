package localdev

import (
	"bytes"
	"context"
	"fmt"
	"github.com/distributed-containers-inc/sanic/util"
	"os"
	"os/exec"
	"text/template"
)

//RegistryNodePort is the port at which the registry is accessible on all of the nodes
// (for performance, connect to this port on specifically the master node from the host)
const RegistryNodePort = 31653

const registryYaml = `
---
kind: Deployment
apiVersion: extensions/v1beta1
metadata:
  name: sanic-registry
  namespace: kube-system
  labels:
    k8s-app: sanic-registry
spec:
  replicas: 1
  selector:
    matchLabels:
      k8s-app: sanic-registry
  template:
    metadata:
      labels:
        k8s-app: sanic-registry
        name: sanic-registry
    spec:
      terminationGracePeriodSeconds: 60
      hostNetwork: true
      nodeSelector:
        node-role.kubernetes.io/master: ""
      tolerations:
      - key: node-role.kubernetes.io/master
        operator: Exists
        effect: NoSchedule
      containers:
      - image: registry:2
        name: registry
        ports:
        - name: registry
          containerPort: 5000

---
kind: Service
apiVersion: v1
metadata:
  name: sanic-registry
  namespace: kube-system
spec:
  selector:
    k8s-app: sanic-registry
  ports:
  - protocol: TCP
    port: 5000
    nodePort: {{.RegistryNodePort}}
  type: NodePort
`

func (provisioner *ProvisionerLocalDev) startRegistry(ctx context.Context) error {
	cmd := exec.Command("kubectl", "apply", "-f", "-")
	cmd.Env = append(os.Environ(), "KUBECONFIG="+provisioner.KubeConfigLocation())

	type yamlConfig struct {
		RegistryNodePort int
	}
	t, err := template.New("").Parse(registryYaml)
	if err != nil {
		panic(err)
	}

	stdinBuffer := &bytes.Buffer{}
	err = t.Execute(stdinBuffer, &yamlConfig{RegistryNodePort: RegistryNodePort})
	if err != nil {
		panic(err)
	}
	cmd.Stdin = stdinBuffer

	errBuffer := &bytes.Buffer{}
	cmd.Stdout = os.Stdout
	cmd.Stderr = errBuffer
	err = cmd.Start()
	if err != nil {
		return err
	}
	err = util.WaitCmdContextually(cmd, ctx)
	if err != nil {
		fmt.Fprint(os.Stderr, errBuffer.String())
	}
	return err
}
