# book-info-operator-demo

## Introduction

This is a go based operator for a simple application. Only two components: ratings service and mysql.

+ ratings: GET and POST to add new data to mysql, deployment and service
+ mysql: pvc, deployment and service

## Prerequirement

```
kubenetes or openshift installed
operator-sdk installed
go installed
git installed
```

## Usage

1. clone this repo

```
git clone git@github.com:AniChikage/bookinfo-operator-demo.git
```

2. get into the folder

```
cd bookinfo-operator-demo
```

3. make specs

```
make generate
make manifests
```

4. check file `Makefile` in the root path, change the `IMG` and `VERSION` to yours, then build and pushs

```
make docker-build docker-push
```

5. install and deploy to your k8s or openshift cluster

```
make install 
make deploy
```

6. check the pods status, you will see `controller` pod is running

```
oc get pods
```

7. apply the config spec to start application

```
oc apply config/samples/bookinfo_v1alpha1_bookinfo.yaml
```

8. check the pods, you will see there are `ratings` and `mysql` pods are running

```
oc get pods
```

9. change the `replicas` to `3`, apply 

```
oc apply config/samples/bookinfo_v1alpha1_bookinfo.yaml
```

10. you will see `ratings` pods will be `3` copies

```
oc get pods
```

