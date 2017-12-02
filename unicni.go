
package main

import (
    "fmt"
    "runtime"
    "net"
    "os"
    "errors"
    "encoding/json"

    "github.com/containernetworking/cni/pkg/types"
    "github.com/containernetworking/cni/pkg/types/current"
    "github.com/containernetworking/cni/pkg/skel"
    "github.com/containernetworking/cni/pkg/version"
    "github.com/containernetworking/plugins/pkg/ns"

    "github.com/vishvananda/netlink"
)

const defaultHost = "127.0.0.1"
const defaultPort = 8080

var (
    ErrLinkNotFound = errors.New("link not found")
)

func init() {
    // This ensures that main runs only on main thread (thread group leader).
    // since namespace ops (unshare, setns) are done for a single thread, we
    // must ensure that the go routine does not jump from OS thread to thread
    runtime.LockOSThread()
}

func cmdAdd(args *skel.CmdArgs) error {
    conf := types.NetConf()
    err := json.Unmarshal(args.StdinData, &conf)
    if err != nil {
        return fmt.Errorf("[UNION CNI]: Failed to load netconf: %v\r\n", err)
    }
   
    // Get annotaions, parse data and control bridge name
    client.CreateInsecureClient(defaultHost, defaultPort)
    parseAnnotation

    result := &current.Result{}

    return types.PrintResult(result, conf.CNIVersion)
}

func cmdDel(args *skel.CmdArgs) error {
    // Delete all port related current user's pod
    return nil
}

func main() {
    skel.PluginMain(cmdAdd, cmdDel, version.All)
}
