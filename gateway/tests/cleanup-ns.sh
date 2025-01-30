#!/bin/bash

# Delete all test namespaces
kubectl get ns -o name | grep "gateway-test-" | xargs -r kubectl delete 