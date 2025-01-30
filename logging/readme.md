```
kro/logging on  main [?] via 💎 v3.0.0 on ☁️  (eu-west-1) on ☁️  xxx@gmail.com 
❯ k get clusterrole | grep logging
cluster-logging                                                        2025-01-28T17:22:49Z
(base) 
kro/logging on  main [?] via 💎 v3.0.0 on ☁️  (eu-west-1) on ☁️  xxx@gmail.com 
❯ k get clusterrolebinding | grep logging
release-name-cluster-logging                                    ClusterRole/cluster-logging                                                 3m18s
(base) 
kro/logging on  main [?] via 💎 v3.0.0 on ☁️  (eu-west-1) on ☁️  xxx@gmail.com 
❯ k get ds                               
NAME            DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE
log-collector   1         1         0       1            0           <none>          3m15s
(base) 
kro/logging on  main [?] via 💎 v3.0.0 on ☁️  (eu-west-1) on ☁️  xxx@gmail.com 
❯ k get sa                               
NAME      SECRETS   AGE
default   0         6m48s
logging   0         3m21s
(base) 
kro/logging on  main [?] via 💎 v3.0.0 on ☁️  (eu-west-1) on ☁️  xxx@gmail.com 
❯ k get cm                               
NAME                    DATA   AGE
kube-root-ca.crt        1      6m57s
log-collector-config    6      3m42s
log-collector-scripts   2      3m39s
(base) 
```
1. Created from the values in your logging instance (law.workspace_id and law.token)
2. Stored in the secret
3. Mounted into the container
4. Read by the reloader scripts
5. Injected into the OUTPUT configuration for Azure Log Analytics

The flow ensures your credentials are securely stored in a Kubernetes secret and properly injected into the Fluent Bit configuration.


---

Looking at the DaemonSet configuration in the resource group, here's how the secret gets mounted:

1. First, it's defined as a volume in the pod spec:
```yaml
volumes:
  - name: law-credentials    # This defines the volume
    secret:
      secretName: logging-law-credentials  # This references our secret
```

2. Then it's mounted in two containers via volumeMounts:

a) In the fluent-bit container:
```yaml
containers:
  - name: fluent-bit
    ...
    volumeMounts:
      - name: law-credentials      # This mounts the volume we defined
        mountPath: /etc/law-credentials  # This is where it's mounted in container
        readOnly: true
```

b) In the config-reloader container:
```yaml
  - name: config-reloader
    ...
    volumeMounts:
      - name: law-credentials      # Same volume mounted here too
        mountPath: /etc/law-credentials  # At the same path
        readOnly: true
```

The flow is:
1. Secret is created with our credentials
2. Pod spec references the secret as a volume named `law-credentials`
3. That volume is mounted at `/etc/law-credentials` in both containers
4. The reloader scripts can then read `/etc/law-credentials/workspaceId` and `/etc/law-credentials/token` to get the values
5. Those values are used to generate the Fluent Bit OUTPUT configuration

Both containers need access because:
- The config-reloader container needs to read the credentials to generate the OUTPUT config
- The fluent-bit container needs to read the credentials to send logs to Azure
