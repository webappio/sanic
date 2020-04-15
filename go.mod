module github.com/distributed-containers-inc/sanic

go 1.12

require (
	github.com/agnivade/levenshtein v1.0.2
	github.com/gdamore/tcell v1.1.2
	github.com/moby/buildkit v0.6.3
	github.com/pkg/errors v0.9.1
	github.com/urfave/cli v1.22.2
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	gopkg.in/yaml.v2 v2.2.4
	sigs.k8s.io/kind v0.3.0
)

replace github.com/containerd/containerd => github.com/containerd/containerd v1.3.1-0.20200227195959-4d242818bf55

replace github.com/docker/docker => github.com/docker/docker v1.4.2-0.20200227233006-38f52c9fec82
