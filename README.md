# Union CNI Plugin
   Implement a simple K8s CNI plugin. 
   This plugin doesn't create the default interface inside the pod, but create others you defined
in pod definition file.

# Example configure
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

# Example YAML
metadata:
  name: Test
  annotations:
     network_info: '{
         "crediential": "user",
         "group": "g1"
     }'

