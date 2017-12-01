
package main

import (
    "fmt"

    "github.com/containernetworking/cni/pkg/skel"
    "github.com/containernetworking/cni/pkg/version"
)

func cmdAdd() {

}

func cmdDel() {

}

func main() {
    skel.PluginMain(cmdAdd, cmdDel, version.All)
}
