package main

import (
    "fmt"
    "github.com/union-cni/pkg/bridge"
)

func main() {
    br, err := bridge.CreateBridge("simple")
    fmt.Printf("create simple bridge: %v %v\r\n", br, err)
}
