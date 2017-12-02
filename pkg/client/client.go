package client

import (
    "fmt"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
 
    "k8s.io/api/core/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
)

type Client struct {
    Host string
    Port string
    Config *rest.Config
}

func CreateInsecureClient(host, port string) *Client {
    client := &Client{
        Host: host,
        Port: port,
    }

    config := &rest.Config{
        Host: fmt.Sprintf("http://%s:%s", host, port),
    } 
   
    config = rest.AnonymousClientConfig(config)
    client.Config = config
  
    return client
}

func (cli *Client) GetAnnotations(namespace, podname string) (*v1.Pod, error) {
    clientset, err := kubernetes.NewForConfig(cli.Config)
    if err != nil {
        return nil, fmt.Errorf("Create client failed: %v", err)
    }

    pod, err := clientset.CoreV1().Pods(namespace).Get(podname, metav1.GetOptions{})
    if err != nil {
        return nil, fmt.Errorf("No such pod %s in namespace %s: %v", podname, namespace, err)
    }

    return pod, nil
}

 
