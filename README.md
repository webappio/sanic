[![Go Report Card](https://goreportcard.com/badge/github.com/distributed-containers-inc/sanic)](https://goreportcard.com/report/github.com/distributed-containers-inc/sanic)

# Sanic Omnitool

Sanic is an all-in-one tool to develop, build, and deploy your Docker/Kubernetes projects. *It focuses on developer experience*:

1. It allows you to volume mount your source code into the containers in real time, so that you have to redeploy less often.
2. It allows you to template your kubernetes configurations based on whatever your team is already comfortable with.
3. It builds things really quickly, so that in the case you do need to build, it's as fast as possible.


A lot of users of Docker/Kubernetes have similar requirements: Build a lot of dockerfiles, template some kubernetes configurations, and then deploy them to a kubernetes cluster.

Each of those steps are currently painful: `docker build` is hard to parallelize well, templates are hard to learn and debug, and local multinode deployment requires lots of internal kubernetes knowledge. 


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


# Requirements

1. [A recent docker client installed, and docker daemon running (i.e., "docker run" should work)](https://docs.docker.com/install/)
2. Access to docker without needing `sudo` every time, e.g.., a sudoers NOPASSWD entry, being in the docker group, or running applicable sanic commands as root.  See [Manage Docker as a non-root user](https://docs.docker.com/install/linux/linux-postinstall/#manage-docker-as-a-non-root-user)
3. [kubectl installed](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

## Getting Started

### Timestamp as a Service
A simple app which consists of three Docker services: A [Python Flask](http://flask.pocoo.org/) api server and web server, and a [Redis](https://redis.io/) container

To try it out:
1. `go get github.com/distributed-containers-inc/sanic`
2. `cd $GOPATH/src/github.com/distributed-containers-inc/sanic`
3. `GO111MODULE=on go install`
4. `cd examples/timestamp-as-a-service`
5. `sanic env dev`
6. `sanic deploy` (to start the local environment, this may take a while. Note the URL printed at the end)
7. `sanic build --push` (to build and push the images)
8. `sanic kubectl get po` (to list the running pods in the new cluster)
9. `sanic kubectl delete po --all` (to force kubernetes to check if new pods have been created, avoiding waiting a minute after building)
10. Navigate to the URL that was printed in step #4 to see the deployed webserver!

### Download
First, install the requirements from the requirements section above.

To install from source, see the Timestamp as a Service example above.

Otherwise, see [the sanic.io downloads page](https://sanic.io/download)

### Configuration
The only configuration file for sanic is the `sanic.yaml` file:
```
# the defined environments -- you should always define at least one
environments:
  # a developer environment, convention is to call it "dev"
  dev:
    # provisioners tell sanic how to push and deploy to a cluster.
    # localdev automatically creates a local 3-node kubernetes cluster with a registry within your docker daemon
    clusterProvisioner: localdev
    # arbitrary shell scripts, defined per-environment
    commands:
      # executed by sanic run do_stuff
    - name: do_stuff
      command: ls -al | awk '{print $1}'
  prod:
    # external points to an existing kubernetes cluster and registry
    clusterProvisioner: external
    clusterProvisionerArgs:
      # registry is either a dockerhub account name, or external registry
      # (it's the prefix of built images)
      registry: registry.company.com
      # edgeNodes are places that ingress controllers are running. This can be left out.
      edgeNodes: sanic.io
      # kubeConfig is a kubectl config that should be used with this cluster
      kubeConfig: ~/.kube/my.prod.config
    commands:
      # notice: commands can be multiline easily with yaml's block syntax
    - name: setup_stuff
      command: |
        ls -al
        pwd
        ps aux

# the global commands block defines commands for every environment.
# note 1: environments can define a command of the same name to override these
# note 2: you must be in an environment (sanic env) to use global commands
commands:
- name: do_stuff
  command: ls -al

# the deploy block tells sanic how to deal with your kubernetes resources
deploy:
  # for the "kustomize" template language, use "distributedcontainers/templater-kustomize"
  # - https://github.com/distributed-containers-inc/sanic-templater-kustomize
  #
  # for the go language templates, use "distributedcontainers/templater-golang" 
  # - https://github.com/distributed-containers-inc/sanic-templater-golang
  #
  # for any other language, feel free to make your own templater image and open an issue to have it included here.
  templaterImage: distributedcontainers/templater-kustomize

# the build block tells sanic how to build your resources
build:
 # ignore directories are specific directories (relative to the directory that contains sanic.yaml)
 # this might also be useful if a directory has thousands of files, to improve build speed (i.e., node_modules)
  ignoreDirs:
  - some/directory
  - node_modules
```

### Pushing
Sanic will automatically push to the registry for the given environment's provisioner if you use `sanic build --push`

Authentication is via `docker login`

### Support / Contact
If you or your team are interested in using sanic, please feel free to schedule a call by emailing [support@sanic.io] -- we're continually adding features based on the needs of our first users.
