package main

import (
    "fmt"
    "os"

    "github.com/union-cni/pkg/client"
)

func main() {
    cli := client.CreateInsecureClient("127.0.0.1", "8080")
    anns, err := cli.GetAnnotations(os.Args[1], os.Args[2])

    fmt.Printf("Annotaions: %v, err %v\r\n", anns, err)
}
