module github.com/distributed-containers-inc/sanic

go 1.12

require (
	github.com/agnivade/levenshtein v1.0.3
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/gdamore/tcell v1.3.0
	github.com/kr/pretty v0.2.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.0.3 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/pkg/errors v0.9.1
	github.com/urfave/cli v1.22.4
	golang.org/x/sync v0.0.0-20200317015054-43a5402ce75a
	golang.org/x/sys v0.0.0-20200413165638-669c56c373c4 // indirect
	golang.org/x/text v0.3.2 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/yaml.v2 v2.2.8
)

replace github.com/containerd/containerd => github.com/containerd/containerd v1.3.1-0.20200227195959-4d242818bf55

replace github.com/docker/docker => github.com/docker/docker v1.4.2-0.20200227233006-38f52c9fec82
