package main

import (
    "fmt"
    "runtime"
    "net"
    "os"
    "encoding/json"

    "github.com/union-cni/pkg/link"
    "github.com/union-cni/pkg/ip"
    "github.com/union-cni/pkg/netinfo"

    "github.com/containernetworking/cni/pkg/skel"
    "github.com/containernetworking/cni/pkg/types"
    "github.com/containernetworking/cni/pkg/types/current"
    "github.com/containernetworking/cni/pkg/version"

    "github.com/vishvananda/netlink"
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

func deleteExternalPorts(netInfo *netinfo.NetworkInfo, netns string) (err error) {
    cred := netInfo.GetCred()
    group := netInfo.GetGroup()
    devID := netInfo.GetDeviceID()
    extPorts := netInfo.GetExternalPorts()
    for _, ext := range extPorts {
        link.DelLinkInNS(ext.ContainerPort, netns)
        extBrName := fmt.Sprintf("%s%s-%s%s", cred, group, devID, ext.ContainerPort)
        link.DeleteBridge(extBrName)
    }
    return
}

func createBridgeMode(cred string, group string, devID string, conPortName string, nspath string) error {
    extBrName := fmt.Sprintf("%s%s-%s%s", cred, group, devID, conPortName)
    br,err := link.CreateBridge(extBrName)
    if err == nil {
        cLink, cHostLink, err := link.CreateVethPairRandom(conPortName)
        if err == nil {
            link.SetPromiscOn(cLink)
            err = link.JoinNetNS(conPortName, nspath)
            if err != nil {
                 fmt.Fprintf(os.Stderr, "[UNION CNI]join nspath %s failed: %v\r\n", nspath, err)
            } else {
                err = br.AddLink(cHostLink)
                if err != nil {
                    link.DelLinkInNS(conPortName, nspath)
                }
            }
        }
    }

    return err
}

func createMacvlanMode(hostPort string, conPort string, mode string, nspath string) error {
    _, err := link.CreateMacvlanInNS(hostPort, conPort, mode, nspath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "[UNION CNI] failed create macvlan port %s: %v\r\n", conPort, err)
    }

    return err
}

func createExternalPorts(netInfo *netinfo.NetworkInfo, nspath string) (err error) {
    cred := netInfo.GetCred()
    group := netInfo.GetGroup()
    devID := netInfo.GetDeviceID()
    extPorts := netInfo.GetExternalPorts()
    for _, ext := range extPorts {
        switch ext.Type { 
            case "macvlan": 
                err = createMacvlanMode(ext.HostPort, ext.ContainerPort, ext.Mode, nspath)
            default:
                err = createBridgeMode(cred, group, devID, ext.ContainerPort, nspath)
        }

        if (err == nil) && (ext.IP != "") {
            err = ip.AddrAddInNS(ext.ContainerPort, ext.IP, nspath)
            fmt.Fprintf(os.Stderr, "[UNION CNI] add ip addr %s: %v\r\n", ext.IP, err)
        }
    }

    fmt.Fprintf(os.Stderr, "[UNION CNI] get extports %v\r\n", extPorts)

    return
}

func appendIntfs(l netlink.Link, nspath string) *current.Interface {
    return link.Interface(l, nspath)
}

func createNetwork(netInfo *netinfo.NetworkInfo, netns string) (*current.Result, error) {
    // assemble result
    result := &current.Result{}

    cred := netInfo.GetCred()
    group := netInfo.GetGroup()
    for chanType, chanName := range netInfo.GetSystemChannels() {
        newBrName := fmt.Sprintf("%s-%s-%s", cred, group, chanType)
        // the length of bridge name must be less than 15 characters.
        if len(newBrName) <= 15 {
            br, err := link.CreateBridge(newBrName)
            if err != nil {
                fmt.Fprintf(os.Stderr, "[UNION CNI] failed to create bridge %s: %v\r\n", newBrName, err)
                return nil, err
            }
            // create veth pairs for channel port
            conLink, hostLink, err := link.CreateVethPairRandom(chanName)
            if err != nil {
                fmt.Fprintf(os.Stderr, "[UNION CNI] failed to create veth pairs %s: %v\r\n", chanName, err)
                return nil, err
            }
            // don't care the promisc mode failed or not
            link.SetPromiscOn(conLink)
            err = link.JoinNetNS(chanName, netns)
            if err != nil {
                fmt.Fprintf(os.Stderr, "[UNION CNI] failed to join namespace %s: %v\r\n", netns, err)
                return nil, err
            }
            err = br.AddLink(hostLink)
            if err != nil {
                fmt.Fprintf(os.Stderr, "[UNION CNI] failed to join bridge %s: %v\r\n", newBrName, err)
                return nil, err
            }

            // append interface results
            result.Interfaces = append(result.Interfaces, link.Interface(br.Data, ""))
            result.Interfaces = append(result.Interfaces, link.Interface(hostLink, ""))
            result.Interfaces = append(result.Interfaces, link.Interface(conLink, netns))
        } else {
            fmt.Fprintf(os.Stderr, "[UNION CNI] bridge name %s is too long\r\n", newBrName)
        }
    }

    // create external ports 
    createExternalPorts(netInfo, netns)

    return result, nil
}

func deleteNetwork(netInfo *netinfo.NetworkInfo, nspath string) error {
    cred := netInfo.GetCred()
    group := netInfo.GetGroup()
    for chanType, chanName := range netInfo.GetSystemChannels() {
        err := link.DelLinkInNS(chanName, nspath)
        fmt.Fprintf(os.Stderr, "[UNION CNI]deleteNetwork %s: %v\r\n", chanName, err)
        sysBr := fmt.Sprintf("%s-%s-%s", cred, group, chanType)
        link.DeleteBridgeIfEmpty(sysBr)
    }

    deleteExternalPorts(netInfo, nspath)

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
