[![Go Report Card](https://goreportcard.com/badge/github.com/distributed-containers-inc/sanic)](https://goreportcard.com/report/github.com/distributed-containers-inc/sanic)

# Sanic Build

Sanic is an all-in-one tool to build, test, and deploy software organized in a [Monorepo](https://en.wikipedia.org/wiki/Monorepo), where:

1. The only things to be built are distinct [Docker](https://www.docker.com/) services with single Dockerfiles
2. Deployment is done with [Kubernetes](https://kubernetes.io/)
3. Unit tests are stored in Dockerfiles in a folder named "dockerfiles" in each service

# Requirements

1. [A recent docker client installed, and docker daemon running (i.e., "docker run" should work)](https://docs.docker.com/install/)
2. Access to docker without needing `sudo` every time, e.g.., a sudoers NOPASSWD entry, being in the docker group, or running applicable sanic commands as root.  See [Manage Docker as a non-root user](https://docs.docker.com/install/linux/linux-postinstall/#manage-docker-as-a-non-root-user)
3. [kubectl installed](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
## Examples

### Timestamp as a Service
A simple app which consists of three Docker services: A [Python Flask](http://flask.pocoo.org/) api server and web server, and a [Redis](https://redis.io/) container

To try it out, clone this repository somewhere, then, in a bash shell:
1. `GO111MODULE=on go install`
2. `cd examples/timestamp-as-a-service`
3. `sanic env dev`
4. `sanic deploy` (to start the local environment, this may take a while. Note the URL printed at the end)
5. `sanic build --push` (to build and push the images)
6. `sanic kubectl get po` (to list the running pods in the new cluster)
7. `sanic kubectl delete po --all` (to force kubernetes to check if new pods have been created, avoiding waiting a minute after building)
8. Navigate to the URL that was printed in step #4 to see the deployed webserver!