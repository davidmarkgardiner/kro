Let me search through the codebase to understand how KRO handles finalizers and cleanup.

Search files...


Based on the search results, I can explain how to properly handle finalizers when cleaning up KRO resources. Here's a comprehensive guide:

### Understanding KRO's Finalizer System

KRO uses finalizers to ensure proper cleanup of resources in a specific order. There are two types of finalizers:
1. ResourceGroup finalizers
2. Instance-specific finalizers

### Normal Cleanup Process

1. When you delete a KRO resource, it will automatically:
   - Delete resources in reverse topological order (to respect dependencies)
   - Remove finalizers once resources are properly cleaned up
   - Handle the complete lifecycle of all related resources

### If Resources Get Stuck in Terminating State

If resources get stuck during deletion, you can manually remove finalizers. Here are the steps:

1. For a KRO instance that's stuck:
```bash
kubectl patch <ResourceType>/<resource-name> -p '{"metadata":{"finalizers":[]}}' --type=merge
```

For example:
```bash
kubectl patch Microservice/test-microservice -p '{"metadata":{"finalizers":[]}}' --type=merge
```

2. For stuck namespaces:
```bash
kubectl get namespace <namespace-name> -o json \
  | jq '.spec.finalizers = []' \
  | kubectl replace --raw "/api/v1/namespaces/<namespace-name>/finalize" -f -
```

### Best Practices for Clean Resource Deletion

1. **Always try normal deletion first**:
```bash
kubectl delete <resource-type> <resource-name>
```

2. **Check resource status** before forcing finalizer removal:
```bash
kubectl get <resource-type> <resource-name> -o yaml
```

3. **Monitor deletion progress** using:
```bash
kubectl get <resource-type> <resource-name> -w
```

4. **Only remove finalizers as a last resort** when:
- The resource is stuck in `Terminating` state
- Normal deletion methods have failed
- You've waited a reasonable time for normal cleanup

5. **Clean up in the correct order**:
   - Delete instances first
   - Then delete ResourceGroups
   - Finally clean up any stuck resources

### Example Cleanup Script

Here's a helpful script you can use to clean up stuck resources:

```bash
#!/bin/bash

RESOURCE_TYPE=$1
RESOURCE_NAME=$2
NAMESPACE=$3

# Try normal deletion first
kubectl delete ${RESOURCE_TYPE} ${RESOURCE_NAME} -n ${NAMESPACE}

# Wait a bit to see if normal deletion works
sleep 10

# Check if resource still exists
if kubectl get ${RESOURCE_TYPE} ${RESOURCE_NAME} -n ${NAMESPACE} &> /dev/null; then
    echo "Resource still exists, removing finalizers..."
    kubectl patch ${RESOURCE_TYPE}/${RESOURCE_NAME} \
        -n ${NAMESPACE} \
        -p '{"metadata":{"finalizers":[]}}' \
        --type=merge
fi
```

### Important Notes

1. **Be Cautious**: Removing finalizers forcefully bypasses normal cleanup procedures. Only do this if you're sure the resources are truly stuck.

2. **Resource Dependencies**: KRO manages complex resource dependencies. Forcing finalizer removal might leave orphaned resources.

3. **Cleanup Order**: Always try to delete resources in the reverse order of their creation to respect dependencies.

4. **Verification**: After removing finalizers, verify that all associated resources are properly cleaned up.

5. **Production Environments**: Be extra careful in production environments. Consider contacting support if resources are consistently getting stuck during deletion.

If you're frequently encountering stuck resources, it might indicate a deeper issue with:
- Resource configurations
- Network connectivity
- Kubernetes cluster health
- Permission issues

In such cases, it's better to investigate and fix the root cause rather than regularly forcing finalizer removal.
