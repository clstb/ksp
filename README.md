[![Build Status](https://ci.clstb.codes/api/badges/clstb/ksp/status.svg)](https://ci.clstb.codes/clstb/ksp)
[![Go Report Card](https://goreportcard.com/badge/github.com/clstb/ksp)](https://goreportcard.com/report/github.com/clstb/ksp)
# KSP - Kubernetes Secret Proxy <!-- omit in toc -->
KSP does annotation based secret data injection and decryption.  

It runs locally and acts as proxy to your kubernetes api server. This means it integrates seamless with `kubectl` and tools that build on it.

#### Table of Contents <!-- omit in toc -->
- [Installation](#installation)
  - [Go](#go)
  - [Binary](#binary)
  - [Docker (In Development)](#docker-in-development)
- [Getting started](#getting-started)
  - [GPG](#gpg)
- [Injectors](#injectors)
  - [GPG](#gpg-1)
    - [Annotations](#annotations)
    - [Notes](#notes)
- [Rationale](#rationale)
  - [POST](#post)
  - [PATCH](#patch)

## Installation

### Go
```sh
export GO111MODULE=on
go get github.com/clstb/ksp
```

### Binary
Download the latest binary [here](https://github.com/clstb/ksp/releases).

### Docker (In Development)
Docker images are available [here](https://hub.docker.com/repository/docker/clstb/ksp).

## Getting started

### GPG
1. Start the ksp proxy with enabled gpg injector.
```sh
ksp proxy --port 8000 --config $HOME/.kube/config --injector-gpg
```
2. Verify `kubectl` is still working.
```
kubectl version
```
3. Encrypt some data.
```
ksp gpg encrypt --keys {YOUR_KEY_ID} --data '{"foo": "bar"}'
```
4. Configure a secret with encrypted data.
```json
{
  "apiVersion": "v1",
  "kind": "Secret",
  "metadata": {
    "name": "example-secret",
    "namespace": "default",
    "annotations": {
      "ksp/inject": "gpg",
    }
  },
  "type": "Opaque",
  "data": {}
}
```
5. Apply it.
```sh
kubectl apply -f example-secret.json
```

## Injectors
Injectors modify secrets passed to them based on annotations.  
They use following interface:
```go
type Injector interface {
    Inject(*corev1.Secret) (*corev1.Secret, error)
}
```

### GPG
The GPG injector decrypts all data fields of the secret using the local gpg cli.

#### Annotations
* `ksp/inject: gpg`

#### Notes
* You can use `ksp gpg encrypt` to encrypt basic JSON files.
* You can encrypt the secret data with multiple public keys. That way it is possible to have a seperate keys for CI/CD or other developers.

## Rationale
Following API endpoints need to be handled:  
* `POST` `/api/v1/namespaces/{namespace}/secrets`
* `PATCH` `/apis/v1/namespaces/{namespace}/secrets/{name}`

### POST
This endpoint handles secret creation.  

The proxy applies following steps:
1. Read secret from request body
2. Call configured injectors with secret
3. Rewrite request body with injected secret
4. Forward request to the kubernetes API server

### PATCH
This endpoint handles secret modification.  
If a secret already exists `kubectl` pulls it from the cluster and computes a diff between the local and cluster state.  
This diff is incorrect because the local state contains encrypted or no secret data.  

The proxy applies following steps to solve this problem:
1. Read patch from request body
2. Retrieve cluster state of the secret
3. Compute local state by patching the cluster state
4. Call configured injectors with secret
5. Compute fixed patch from injected secret and cluster state
6. Rewrite request body with fixed patch
7. Forward request to kubernetes API server