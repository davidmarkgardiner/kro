```
kro/logging on  main [?] via 💎 v3.0.0 on ☁️  (eu-west-1) on ☁️  davidmarkgardiner@gmail.com 
❯ k get clusterrole | grep logging
cluster-logging                                                        2025-01-28T17:22:49Z
(base) 
kro/logging on  main [?] via 💎 v3.0.0 on ☁️  (eu-west-1) on ☁️  davidmarkgardiner@gmail.com 
❯ k get clusterrolebinding | grep logging
release-name-cluster-logging                                    ClusterRole/cluster-logging                                                 3m18s
(base) 
kro/logging on  main [?] via 💎 v3.0.0 on ☁️  (eu-west-1) on ☁️  davidmarkgardiner@gmail.com 
❯ k get ds                               
NAME            DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE
log-collector   1         1         0       1            0           <none>          3m15s
(base) 
kro/logging on  main [?] via 💎 v3.0.0 on ☁️  (eu-west-1) on ☁️  davidmarkgardiner@gmail.com 
❯ k get sa                               
NAME      SECRETS   AGE
default   0         6m48s
logging   0         3m21s
(base) 
kro/logging on  main [?] via 💎 v3.0.0 on ☁️  (eu-west-1) on ☁️  davidmarkgardiner@gmail.com 
❯ k get cm                               
NAME                    DATA   AGE
kube-root-ca.crt        1      6m57s
log-collector-config    6      3m42s
log-collector-scripts   2      3m39s
(base) 
```
