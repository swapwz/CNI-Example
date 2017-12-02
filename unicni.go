package main

import (
    "fmt"
    "runtime"
    "net"
    "os"
    "errors"
    "encoding/json"

    "github.com/union-cni/pkg/client"

    "github.com/containernetworking/cni/pkg/skel"
    "github.com/containernetworking/cni/pkg/types"
    "github.com/containernetworking/cni/pkg/types/current"
    "github.com/containernetworking/cni/pkg/version"
)

const (
    defaultHost = "127.0.0.1"
    defaultPort = "8080"
)

var (
    ErrLinkNotFound = errors.New("link not found")
)


type K8SArgs struct {
        types.CommonArgs
        IP                         net.IP
        K8S_POD_NAME               types.UnmarshallableString
        K8S_POD_NAMESPACE          types.UnmarshallableString
        K8S_POD_INFRA_CONTAINER_ID types.UnmarshallableString
}

func init() {
    // This ensures that main runs only on main thread (thread group leader).
    // since namespace ops (unshare, setns) are done for a single thread, we
    // must ensure that the go routine does not jump from OS thread to thread
    runtime.LockOSThread()
}

func cmdAdd(args *skel.CmdArgs) error {
    conf := types.NetConf{}
    err := json.Unmarshal(args.StdinData, &conf)
    if err != nil {
        fmt.Fprintf(os.Stderr, "[UNION CNI] failed to load netconf: %v", err)
        return err
    }

    k8sArgs := K8SArgs{}
    err = types.LoadArgs(args.Args, &k8sArgs)
    if err != nil {
        fmt.Fprintf(os.Stderr, "[UNION CNI]: Failed to load args %q: %v\r\n", args.Args, err)
        return err
    }
   
    // Get annotaions, parse data and control bridge name
    fmt.Fprintf(os.Stderr, "[UNION CNI] k8s namespace: %s, pod name: %s\r\n", k8sArgs.K8S_POD_NAMESPACE, k8sArgs.K8S_POD_NAME)
    k8scli := client.CreateInsecureClient(defaultHost, defaultPort)
    pod, err := k8scli.GetPod(string(k8sArgs.K8S_POD_NAMESPACE), string(k8sArgs.K8S_POD_NAME))
    if err != nil {
        fmt.Fprintf(os.Stderr, "[UNION CNI] failed to get pod: %v\r\n", err)
        return err
    }
    fmt.Fprintf(os.Stderr, "[UNION CNI] find pod %v", pod)
    //parseDataControlName()

    result := &current.Result{}
    return types.PrintResult(result, conf.CNIVersion)
}

func cmdDel(args *skel.CmdArgs) error {
    // Delete all port related current user's pod
    fmt.Fprintf(os.Stderr, "[UNION CNI] action delete.\r\n")
    return nil
}

func main() {
    skel.PluginMain(cmdAdd, cmdDel, version.All)
}
