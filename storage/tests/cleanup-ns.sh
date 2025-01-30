#!/bin/bash

# Get all namespaces in Terminating state
TERMINATING_NS=$(kubectl get ns | grep Terminating | awk '{print $1}')

for ns in $TERMINATING_NS; do
    echo "Removing finalizers from namespace: $ns"
    # Get the namespace manifest and remove finalizers
    kubectl get namespace $ns -o json | jq '.spec.finalizers = []' | kubectl replace --raw "/api/v1/namespaces/$ns/finalize" -f -
done