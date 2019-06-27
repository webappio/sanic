[![Go Report Card](https://goreportcard.com/badge/github.com/distributed-containers-inc/sanic)](https://goreportcard.com/report/github.com/distributed-containers-inc/sanic)

# Sanic Omnitool

Sanic is an all-in-one tool to develop, build, and deploy your Docker/Kubernetes projects.

## Why?

### Why Sanic
A lot of users of Docker/Kubernetes have similar requirements: Build a lot of dockerfiles, template some kubernetes configurations, and then deploy them to a kubernetes cluster.

Each of those steps are currently painful: `docker build` is hard to parallelize well, templates are hard to learn and debug, and local multinode deployment requires lots of internal kubernetes knowledge. 

*Sanic focuses on developer experience*:
1. It volume mounts your source code into the containers in real time, so that you have to redeploy less often.
2. It allows you to template your kubernetes configurations based on whatever your team is already comfortable with.
3. It builds things really quickly, so that in the case you do need to build, it's as fast as possible.

#### Concurrent builds
Sanic discovers all Dockerfiles in your repository, and builds them in parallel using [buildkit](https://github.com/moby/buildkit).  This allows it to build incredibly quickly, and share layers across dockerfiles with ease.

It also generates a unique tag for every build, so that you can follow best practices and avoid using `:latest`.


#### Live-Mounting
Sanic allows you to mount your source code inside of the containers running it in the `localdev` environment.

The templater is run with the `PROJECT_DIR` environment variable set to the location of the project, so you can create a Kubernetes Volume from `$PROJECT_DIR/services/web` to `app/` and then enable source code reloading.

This allows you to overwrite the contents of the Dockerfile with your actual source code, so that changes immediately propagate, and you don't need to build/deploy after every change.


#### Templating
We believe that developers shouldn't have to learn a new templating language for every tool.  If you use Mako for your webserver, you should have web.yaml.mako to generate your kubernetes configuration.  This lets new developers ramp up faster.

If your templating language isn't supported, you can create a new image and sanic will use it with ease! See [sanic-templater-golang](https://github.com/distributed-containers-inc/sanic-templater-golang) for an example.

Built templates go into an /out folder, so if there are any errors, it's easy to see exactly where they are.

### Why Docker
It's easy enough to deploy a static website without docker, but deploying apps without Docker causes huge headaches:
- Mismatched JDK versions break Java apps
- Missing VirtualEnvs break Python apps
- Differing gcc versions make it hard to build C/C++/other compiled apps (you need a file that says "how to build"!)

Docker solves all of these things, at the expense of disk space -- it's the logical choice to build & run services

### Why Kubernetes
Docker itself is fine for single machines, but has very few features for running apps across multiple servers. It also lacks the ability to re-schedule crashed containers easily.  Kubernetes provides lots of abstractions over containers to let you say "I want 3 API servers running on 3 different machines, and a load balancer to select which ones are up"

Another benefit is that Kubernetes is a decentralized analogue of Amazon Web Services, you can run it on premises, without internet, and change providers based on your needs.

# Requirements

1. [A recent docker client installed, and docker daemon running (i.e., "docker run" should work)](https://docs.docker.com/install/)
2. Access to docker without needing `sudo` every time, e.g.., a sudoers NOPASSWD entry, being in the docker group, or running applicable sanic commands as root.  See [Manage Docker as a non-root user](https://docs.docker.com/install/linux/linux-postinstall/#manage-docker-as-a-non-root-user)
3. [kubectl installed](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
## Getting Started

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

Read more about this example in the `guides/BareMetalProduction.md` guide.  In particular, the configuration is explained there.
