[![Go Report Card](https://goreportcard.com/badge/github.com/distributed-containers-inc/sanic)](https://goreportcard.com/report/github.com/distributed-containers-inc/sanic)

# Sanic Build

Sanic is an all-in-one tool to build, test, and deploy software organized in a [Monorepo](https://en.wikipedia.org/wiki/Monorepo), where:

1. The only things to be built are distinct [Docker](https://www.docker.com/) services with single Dockerfiles
2. Deployment is done with [Kubernetes](https://kubernetes.io/)
3. Unit tests are stored in Dockerfiles in a folder named "dockerfiles" in each service

## Examples

### Timestamp as a Service
A simple app which consists of three Docker services: A [Python Flask](http://flask.pocoo.org/) api server and web server, and a [Redis](https://redis.io/) container