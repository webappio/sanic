package util

import (
	"bytes"
	"context"
	"fmt"
	"github.com/layer-devops/sanic/pkg/provisioners/provisioner"
	"github.com/layer-devops/sanic/pkg/util"
	"github.com/pkg/errors"
	"os"
	"text/template"
)

//RegistryNodePort is the port at which the registry is accessible on all of the nodes
// (for performance, connect to this port on specifically the master node from the host)
const registryYaml = `
---
kind: Deployment
apiVersion: apps/v1
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
      terminationGracePeriodSeconds: 10
      nodeSelector:
        {{range $key, $value := .NodeSelectors}}{{$key}}: "{{$value}}"
{{end}}
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
`

//StartRegistry : makes a pod definition using registry:2
func StartRegistry(provisioner provisioner.Provisioner, ctx context.Context, nodeSelectors map[string]string) error {
	cmd, err := provisioner.KubectlCommand("apply", "-f", "-")

	if err != nil {
		return errors.Wrap(err, "could not start the registry for this environment")
	}

	type yamlConfig struct {
		RegistryNodePort int
		NodeSelectors    map[string]string
	}
	t, err := template.New("").Parse(registryYaml)
	if err != nil {
		panic(err)
	}

	stdinBuffer := &bytes.Buffer{}
	err = t.Execute(stdinBuffer, &yamlConfig{NodeSelectors: nodeSelectors})
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
	err = util.WaitCmdContextually(ctx, cmd)
	if err != nil {
		fmt.Fprint(os.Stderr, errBuffer.String())
	}
	return err
}
