# kone [![Build Status](https://travis-ci.com/IBM/kone.svg?branch=master)](https://travis-ci.com/IBM/kone)

`kone` is a tool for building and deploying nodejs applications to Kubernetes.

`kone` is basically [`ko`](https://github.com/google/ko) for Node.js.

Disclaimer: this is a prototype. Use at your own risk.

## Installation

You need to have `go`. Then `kone` can be installed via:

```shell
go get github.com/ibm/kone/cmd/kone
```

To update your installation:

```shell
go get -u github.com/ibm/kone/cmd/kone
```

## The `kone` Model

**One of the goals of `kone` is to make containers invisible infrastructure.**
Simply replace image references in your Kubernetes yaml with the path for
your Node.js application, and `kone` will handle containerizing and publishing that
container image as needed.

For example, you might use the following in a Kubernetes `Deployment` resource:

```yaml
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: hello-world
spec:
  selector:
    matchLabels:
      foo: bar
  replicas: 1
  template:
    metadata:
      labels:
        foo: bar
    spec:
      containers:
      - name: hello-world
        # This is the relative path for the Node.js application to containerize and run.
        image: ../my-node-app
        ports:
        - containerPort: 8080
```

### Determining supported node app

`kone` looks for `package.json` under the path specified in the YAML file.
Relative paths are resolved from the YAML file location.

If found, `kone` checks:
- the package name is set in `package.json` and use it as base image name
- `main.js` exists, and use it as the docker entry point `ENTRYPOINT ["node", "main.js"]`

If one of the conditions is not met, `kone` skips the package.

### Results

Employing this convention enables `kone` to have effectively zero configuration
and enable very fast development iteration.

## Usage

`kone` has four commands, most of which build and publish images as part of
their execution.  By default, `kone` publishes images to a Docker Registry
specified via `KO_DOCKER_REPO`.

However, these same commands can be directed to operate locally as well via
the `--local` or `-L` command (or setting `KO_DOCKER_REPO=ko.local`).  See
the [`minikube` section](./README.md#with-minikube) for more detail.

### `kone publish`

`kone publish` simply builds and publishes images for each path passed as
an argument. It prints the images' published digests after each image is published.

```shell
$ kone publish ./nodejs
2019/08/13 16:46:38 Using base index.docker.io/library/node:lts-slim for ./nodejs
2019/08/13 16:46:39 Publishing index.docker.io/villardl/filter-dispatcher-197beaeba3358032d029cb68f14adaf6:latest
2019/08/13 16:46:39 mounted blob: sha256:93f92d87eaa35181f944eee8bf9e240c09fd5b21b78ee865dd77676f5d646e9d
2019/08/13 16:46:39 mounted blob: sha256:a9d2af5037827579dd8de01460594e7ccaceff88a1381abf7e092f2f68329e31
2019/08/13 16:46:39 mounted blob: sha256:0a4690c5d889e116874bf45dc757b515565a3bd9b0f6c04054d62280bb4f4ecf
2019/08/13 16:46:39 mounted blob: sha256:68e3f078a91d377648dce9ab8f6a2fcf38164be1cc52eb1a6b4d9b633260cd79
2019/08/13 16:46:39 mounted blob: sha256:0cde646bed8c17ceb3ca87cdf65c9d25edfb803d0802fcac62f86b1e55d568e5
2019/08/13 16:46:39 existing blob: sha256:e4c49295e9e249983b0adee0ff7a643ee4248115d1245870b215d8ded74ef553
2019/08/13 16:46:40 pushed blob: sha256:5a0f660db6333b9bb7e46cf31a110741b97ab6c23c977e5e22b7f96cbbcf69a7
2019/08/13 16:46:40 index.docker.io/villardl/filter-dispatcher-197beaeba3358032d029cb68f14adaf6:latest: digest: sha256:798e1d12b44c5cae81d8b36c913461299b19248b2ddd1fb206c882fef44c5939 size: 1242
2019/08/13 16:46:40 Published index.docker.io/villardl/filter-dispatcher-197beaeba3358032d029cb68f14adaf6@sha256:798e1d12b44c5cae81d8b36c913461299b19248b2ddd1fb206c882fef44c5939
index.docker.io/villardl/filter-dispatcher-197beaeba3358032d029cb68f14adaf6@sha256:798e1d12b44c5cae81d8b36c913461299b19248b2ddd1fb206c882fef44c5939
```

### `kone resolve`

`kone resolve` takes Kubernetes yaml files in the style of `kubectl apply`
and (based on the [model above](#the-kone-model)) determines the set of
paths to containerize, and publish.

The output of `kone resolve` is the concatenated yaml with paths
replaced with published image digests. Following the example above,
this would be:

```shell
# Command
export KO_DOCKER_REPO="docker.io/<your-id>"
ko resolve -f deployment.yaml

# Output
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: hello-world
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: hello-world
        # This is the digest of the published image containing the go binary.
        image: docker.id/your-id/filter-dispatcher@sha256:5755af3776d9dbf206f1d9d29e0319887771e2c0e4c58e469d4a55b88d89de51
        ports:
        - containerPort: 8080
```

`kone resolve`, `kone apply`, and `kone create` accept an optional `--selector` or `-l`
flag,  similar to `kubectl`, which can be used to filter the resources from the
input Kubernetes YAMLs by their `metadata.labels`.

In the case of `kone resolve`, `--selector` will render only the resources that are selected by the provided selector.

See [the documentation on Kubernetes selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) for more information on using label selectors.

### `kone apply`

`kone apply` is intended to parallel `kubectl apply`, but acts on the same
resolved output as `kone resolve` emits. It is expected that `kone apply` will act
as the vehicle for rapid iteration during development. As changes are made to a
particular application, you can run: `kone apply -f unit.yaml` to rapidly
repush, and redeploy their changes.

`kone apply` will invoke `kubectl apply` under the hood, and therefore apply
to whatever `kubectl` context is active.

### `kone delete`

`kone delete` simply passes through to `kubectl delete`. It is exposed purely out
of convenience for cleaning up resources created through `kone apply`.

### `kone version`

`kone version` prints version of kone. For not released binaries it will print hash of latest commit in current git tree.

## With `minikube`

You can use `kone` with `minikube` via a Docker Registry, but this involves
publishing images only to pull them back down to your machine again.  To avoid
this, `kone` exposes `--local` or `-L` options to instead publish the images to
the local machine's Docker daemon.

This would look something like:

```shell
# Use the minikube docker daemon.
eval $(minikube docker-env)

# Make sure minikube is the current kubectl context.
kubectl config use-context minikube

# Deploy to minikube w/o registry.
kone apply -L -f config/

# This is the same as above.
KO_DOCKER_REPO=ko.local kone apply -f config/
```

A caveat of this approach is that it will not work if your container is
configured with `imagePullPolicy: Always` because despite having the image
locally, a pull is performed to ensure we have the latest version, it still
exists, and that access hasn't been revoked. A workaround for this is to
use `imagePullPolicy: IfNotPresent`, which should work well with `kone` in
all contexts.

Images will appear in the Docker daemon as `ko.local/import.path.com/foo/cmd/bar`.

## Configuration via `.ko.yaml`

While `kone` aims to have zero configuration, there are certain scenarios where
you will want to override `kone`'s default behavior. This is done via `.ko.yaml`.

`.ko.yaml` is put into the directory from which `kone` will be invoked. One can
override the directory with the `KO_CONFIG_PATH` environment variable.

If neither is present, then `kone` will rely on its default behaviors.

### Overriding the default base image

By default, `kone` makes use of `docker.io/node/node:lts-slim` as the base image
for containers. There are a wide array of scenarios in which overriding this
makes sense, for example:
1. Pinning to a particular digest of this image for repeatable builds,
1. Replacing this streamlined base image with another with better debugging
  tools (e.g. a shell, like `docker.io/node/node:lts`).

The default base image `kone` uses can be changed by simply adding the following
line to `.ko.yaml`:

```yaml
defaultBaseImage: docker.io/node/node:<another-tag>
```

# Overriding the base for particular package

Some of your node packages may have requirements that are a more unique,
and you may want to direct `kone` to use a particular base image for just those packages.

The base image `kone` uses can be changed by adding the following to `package.json`:

```json
{
  "kone": {
    "defaultBaseImage": "github.com/my-org/my-repo/path/to/binary: docker.io/another/base:latest"
  }
}
```

### Why isn't `KO_DOCKER_REPO` part of `.ko.yaml`?

Once introduced to `.ko.yaml`, you may find yourself wondering: Why does it
not hold the value of `$KO_DOCKER_REPO`?

The answer is that `.ko.yaml` is expected to sit in the root of a repository,
and get checked in and versioned alongside your source code. This also means
that the configured values will be shared across developers on a project, which
for `KO_DOCKER_REPO` is actually undesirable because each developer is (likely)
using their own docker repository and cluster.


## Including static assets

`kone` includes all assets in your node root directory.

## Enable Autocompletion

To generate an bash completion script, you can run:
```
kone completion
```

To use the completion script, you can copy the script in your bash_completion directory (e.g. /usr/local/etc/bash_completion.d/):
```
kone completion > /usr/local/etc/bash_completion.d/kone
```
 or source it in your shell by running:
```
source <(kone completion)
```

## Relevance to Release Management

`kone` is also useful for helping manage releases. For example, if your project
periodically releases a set of images and configuration to launch those images
on a Kubernetes cluster, release binaries may be published and the configuration
generated via:

```shell
export KO_DOCKER_REPO="docker.io/<your-id>"
kone resolve -f config/ > release.yaml
```

This will publish all of the components as container images to
`docker.io/<your-id>/...` and create a `release.yaml` file containing all of the
configuration for your application with inlined image references.

This resulting configuration may then be installed onto Kubernetes clusters via:

```shell
kubectl apply -f release.yaml
```

## Acknowledgements

This work 99% [ko](https://github.com/google/ko).
