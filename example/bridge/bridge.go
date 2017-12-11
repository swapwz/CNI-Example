package main

import (
    "fmt"
    "github.com/union-cni/pkg/link"
)

func main() {
    br, err := link.CreateBridge("simple")
    fmt.Printf("create simple bridge: %v %v\r\n", br, err)
    link.DeleteBridgeIfEmpty("./my")
}
