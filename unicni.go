package main

import (
    "fmt"
    "runtime"
    "net"
    "os"
    "encoding/json"

    "github.com/union-cni/pkg/bridge"
    "github.com/union-cni/pkg/veth"
    "github.com/union-cni/pkg/netinfo"

    "github.com/containernetworking/cni/pkg/skel"
    "github.com/containernetworking/cni/pkg/types"
    "github.com/containernetworking/cni/pkg/types/current"
    "github.com/containernetworking/cni/pkg/version"
    "github.com/containernetworking/plugins/pkg/ns"
)

const (
    defaultHost = "127.0.0.1"
    defaultPort = "8080"
    credAnnotation = "credential"
    groupAnnotation = "group"
    ctrlPortAnnotation = "control_port"
    dataPortAnnotation = "data_port"
) 

type CNINetConf struct {
    types.NetConf
    KubeMaster string `json:"kubemaster"`
}

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

func createNetwork(net *netinfo.NetworkInfo, netns string) (*current.Result, error) {
    // the length of bridge name must be less than 15 characters.
    ctrlBrName := net.GetCtrlBridgeName()
    ctrlBr, err := bridge.CreateBridge(ctrlBrName)
    if err != nil {
         fmt.Fprintf(os.Stderr, "[UNION CNI] failed to create bridge %s: %v\r\n", ctrlBrName, err)
         return nil, err
    }

    dataBrName := net.GetDataBridgeName()
    dataBr, err := bridge.CreateBridge(dataBrName)
    if err != nil {
         fmt.Fprintf(os.Stderr, "[UNION CNI] failed to create bridge %s: %v\r\n", ctrlBrName, err)
         return nil, err
    }

    // create veth pairs
    ctrlPort := net.GetCtrlPort()
    ctrlLink, ctrlHostLink, err := veth.CreateVethPairRandom(ctrlPort)
    if err != nil {
        fmt.Fprintf(os.Stderr, "[UNION CNI] failed to create veth pairs %s: %v\r\n", ctrlPort, err)
        return nil, err
    }

    err = veth.JoinNetNS(ctrlPort, netns)
    if err != nil {
        fmt.Fprintf(os.Stderr, "[UNION CNI] failed to join namespace %s: %v\r\n", netns, err)
        return nil, err
    }

    err = ctrlBr.AddLink(ctrlHostLink)
    if err != nil {
        fmt.Fprintf(os.Stderr, "[UNION CNI] failed to join bridge %s: %v\r\n", ctrlBr.Name, err)
        return nil, err
    }

    dataPort := net.GetDataPort()
    dataLink, dataHostLink, err := veth.CreateVethPairRandom(dataPort)
    if err != nil {
        fmt.Fprintf(os.Stderr, "[UNION CNI] failed to create veth pairs %s: %v\r\n", dataPort, err)
        return nil, err
    }

    err = veth.JoinNetNS(dataPort, netns)
    if err != nil {
        fmt.Fprintf(os.Stderr, "[UNION CNI] failed to join namespace %s: %v\r\n", netns, err)
        return nil, err
    }

    err = dataBr.AddLink(dataHostLink)
    if err != nil {
        fmt.Fprintf(os.Stderr, "[UNION CNI] failed to join bridge %s: %v\r\n", dataBr.Name, err)
        return nil, err
    }

    // assemble result
    result := &current.Result{}
    ctrlBrIntf := ctrlBr.BridgeInterface()
    dataBrIntf := dataBr.BridgeInterface()
    netNS, err := ns.GetNS(netns)
    ctrlLinkIntf := veth.VethInterface(ctrlLink, netNS)
    dataLinkIntf := veth.VethInterface(dataLink, netNS)
    netNS.Close() 
    curNS, _ := ns.GetCurrentNS()
    ctrlHostIntf := veth.VethInterface(ctrlHostLink, curNS)
    dataHostIntf := veth.VethInterface(dataHostLink, curNS)

    result.Interfaces = []*current.Interface{ctrlBrIntf, dataBrIntf, 
        ctrlLinkIntf, ctrlHostIntf, dataLinkIntf, dataHostIntf}

    return result, nil
}

func deleteNetwork(netInfo *netinfo.NetworkInfo, nspath string) error {
    ctrlPortName := netInfo.GetCtrlPort()
    err := veth.DelLink(ctrlPortName, nspath)
    if err != nil && err != veth.ErrLinkNotFound {
         fmt.Fprintf(os.Stderr, "[UNION CNI] failed to delete link %s: %v\r\n", ctrlPortName, err)
         return err
    }

    dataPortName := netInfo.GetDataPort()
    err = veth.DelLink(dataPortName, nspath)
    if err != nil && err != veth.ErrLinkNotFound {
         fmt.Fprintf(os.Stderr, "[UNION CNI] failed to delete link %s: %v\r\n", ctrlPortName, err)
         return err
    }
    return nil
}

func cmdAdd(args *skel.CmdArgs) error {
    conf := CNINetConf{}
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
    result := &current.Result{}
    fmt.Fprintf(os.Stderr, "[UNION CNI] k8s namespace: %s, pod name: %s\r\n", k8sArgs.K8S_POD_NAMESPACE, k8sArgs.K8S_POD_NAME)
    if len(k8sArgs.K8S_POD_NAME) != 0 || len(k8sArgs.K8S_POD_NAMESPACE) != 0 {
        var kubeMaster string
        if conf.KubeMaster != "" {
             kubeMaster = conf.KubeMaster
        } else {
	     kubeMaster = defaultHost
        }
        fmt.Fprintf(os.Stderr, "[UNION CNI] kubemaster %v\r\n", kubeMaster)
        netInfo, _ := netinfo.GetNetInfo(kubeMaster, defaultPort, 
                            string(k8sArgs.K8S_POD_NAMESPACE), 
                            string(k8sArgs.K8S_POD_NAME))
        // If no annotaion, just ignore it.
        if netInfo != nil {
            result,_ = createNetwork(netInfo, args.Netns) 
        }
    }
    return types.PrintResult(result, conf.CNIVersion)
}

func cmdDel(args *skel.CmdArgs) error {
    // Delete all port related current user's pod
    fmt.Fprintf(os.Stderr, "[UNION CNI] action delete.\r\n")
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

    // Get annotaions, parse data and control bridge name, and delete all
    fmt.Fprintf(os.Stderr, "[UNION CNI] k8s namespace: %s, pod name: %s\r\n", k8sArgs.K8S_POD_NAMESPACE, k8sArgs.K8S_POD_NAME)
    if len(k8sArgs.K8S_POD_NAME) != 0 || len(k8sArgs.K8S_POD_NAMESPACE) != 0 {
        netInfo, err := netinfo.GetNetInfo(defaultHost, defaultPort, string(k8sArgs.K8S_POD_NAMESPACE), string(k8sArgs.K8S_POD_NAME))
        if err == nil {
             deleteNetwork(netInfo, args.Netns)
        }
    }
    return nil
}

func main() {
    skel.PluginMain(cmdAdd, cmdDel, version.All)
}
