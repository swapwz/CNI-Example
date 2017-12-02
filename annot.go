package main

import (
    "fmt"

    "github.com/union-cni/pkg/client"
)

func main() {
    cli := client.CreateInsecureClient("127.0.0.1", "80")
    anns, err := cli.GetAnnotations("default", "v9-simware")

    fmt.Printf("Annotaions: %v, err %v\r\n", anns, err)
}
