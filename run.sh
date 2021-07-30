#!/bin/bash

make generate

echo "-----------generate done--------------"

make manifests

echo "-----------manifests done--------------"

make docker-build docker-push 

echo "-----------docker done--------------"

make install

echo "-----------install done--------------"

make deploy

echo "-----------deploy done--------------"
