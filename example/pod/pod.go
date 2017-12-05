package main

import (
    "fmt"
    "os"

    "github.com/union-cni/pkg/client"
)

func main() {
    cli := client.CreateInsecureClient("127.0.0.1", "8080")
    pod, err := cli.GetPod(os.Args[1], os.Args[2])
    if err != nil {
        fmt.Printf("no such pod")
        return
    }
    fmt.Printf("Annotation %q, err %v\r\n", pod.Annotations, err)
}
