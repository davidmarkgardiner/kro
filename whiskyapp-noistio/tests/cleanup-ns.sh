#!/bin/bash

# This script cleans up test namespaces that might be left over from failed tests

# Find all test namespaces
for ns in $(kubectl get ns -o name | grep test-whiskyapp); do
    echo "Cleaning up namespace: $ns"
    kubectl patch $ns -p '{"metadata":{"finalizers":[]}}' --type=merge
    kubectl delete $ns --wait=false
done 