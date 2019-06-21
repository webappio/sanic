# WIP
This guide is a work in progress. The following things need to be finished:

- [ ] Ingress configuration / explanation
- [ ] Master Taints / explanation
- [ ] Modifying sanic.yaml to deploy
- [ ] Walkthrough on fresh client / server install to make sure nothing is missing

# Example for deploying to a generic server running ubuntu 18.04

This guide will show you the exact steps required to develop & deploy a multi-image app to a kubernetes cluster hosted on a dedicated server provider
In particular, it will deploy the one in example/timestamp-as-a-service


## Server configuration
1. ssh to your server (username / password / hostname depends on which provider you use)
`ssh username@1.2.3.4`

2. On the server (i.e., the terminal from step 1), [install Docker, configured to work with Kubernetes](https://kubernetes.io/docs/setup/production-environment/container-runtimes/#docker)
3. On the server, [install kubeadm: A tool to set up kubernetes](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/install-kubeadm/#installing-kubeadm-kubelet-and-kubectl)
4. On the server, `kubeadm init --pod-network-cidr=10.244.0.0/16`
5. On the server, `KUBECONFIG=/etc/kubernetes/admin.conf kubectl apply -f https://docs.projectcalico.org/v3.7/manifests/canal.yaml`
6. On the server, `KUBECONFIG=/etc/kubernetes/admin.conf kubectl apply -f https://github.com/distributed-containers-inc/sanic-site/blob/0ef7f3b9ad234f88ea9b1e1ac169fb17400e42f7/hack/bare-metal-nginx-ingress.yaml` 
7. On the server, `KUBECONFIG=/etc/kubernetes/admin.conf kubectl taint nodes --all node-role.kubernetes.io/master-` 
## Developer PC configuration
Now that we've configured the server, we have to set up our local computer.

### Docker
Your computer (not the server!) needs docker installed as well, since we build locally. Follow the official [Docker Install steps](https://docs.docker.com/install/)

### Registry
Sanic allows you to use any registry, but the easiest one to use is [Docker Hub](https://hub.docker.com/). Create an account there and verify your email. Keep track of the username & password you chose.

After setting up the registry, log in with `docker login -u (the username)`, sanic will pull the credentials from here when we are building 

### Kubectl
To administer any kubernetes cluster, you need kubectl installed on your local computer. Follow [install kubectl steps](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

Back on your host, authenticate to the kubernetes cluster: `scp username@1.2.3.4:/etc/kubelet/admin.conf ~/.sanic-example.conf`

At this point, you should be able to `KUBECONFIG=~/.sanic-example.conf kubectl get services`


## Exploring timestamp-as-a-service

The timestamp-as-a-service example contains three services running a Python webserver, api server, and a redis instance.  It's reasonably close to what a small SaaS company might use.

Sanic uses the directory which contains the sanic.yaml file as the project name, so its project root is in `timestamp-as-a-service`

At build time, Sanic recursively discovers all directories which contain `Dockerfile`s and builds them

The `sanic.yaml` contains two environments: These are essentially places you can push from/to, and define commands on.

Commands can simply be run with `sanic run print_dev` if in the `dev` environment.

The `dev` environment has its `clusterProvisioner` key set to `localdev` -- this means that when you `sanic deploy` for the first time, Sanic will set up a 3 node kubernetes cluster on your local computer and deploy there

The `prod` environment has its `clusterProvisioner` key set to `external` -- this means that `sanic deploy` will not be in charge of starting the kubernetes cluster, it will just deploy the kubernetes resources using the given settings
- `registry` can be a URL (for a private registry) or a docker hub username, the credentials come from `docker login`
- `edgeNodes` are the hosts running Master nodes (this depends on the provider, but you can generally find these with `kubectl get node -o wide`)
- `kubeConfig` is the location of the KUBE_CONFIG file, if you can "kubectl", try `echo ${KUBECONFIG:-~/.kube/config}` to find it

The `deploy` key tells sanic where to find the templates (in this case, `deploy/in`) and which templater image to use. In this case, one that uses Go template syntax for simplicity, but in practice this should be whatever templating language you use for your webserver.