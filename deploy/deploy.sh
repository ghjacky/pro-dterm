#!/bin/bash
kubectl delete -f deployment.yaml
kubectl create -f deployment.yaml