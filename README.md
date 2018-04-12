Uinion CNI Plugin
---

# Introduction

It implements a simple K8s CNI plugin. 
This plugin doesn't create the default interface inside the pod, but create others you defined in pod definition file.

# How to use it?

- checkout the source code into your linux system. 
> `# git clone https://github.com/swapwz/union-cni.git`
- use go compiler to build the binary
> `# go build unicni`
- put the binary into your CNI path, default is /opt/cni/bin
> `# cp unicni /opt/cni/bin/ `
- write your own configuration, with the file /etc/cni/net.d/00-unicni
```
    uni.conflist:
    {
        "cniVersion": "0.3.1",
        "name": "union-net", 
        "plugins": [
            {
                "type": "unicni"
                "kubemaster": "127.0.0.1"
            },
            {
                "type": "bridge",
                "bridge": "cni0",
                "ipam": {
                    "type": "host-local",
                    "subnet": "10.19.0.0/16",
                    "gateway": "10.19.0.1"
                }
            }
         ]
    }
```

## The YAML Example 
```
metadata:
  name: Test
  annotations:
     network_info: '{
         "crediential": "user",
         "group": "g1"
     }'
```

# Implementation

To be continue
