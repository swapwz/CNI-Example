package netinfo

import (
    "fmt"
    "os"
    "errors"
    "encoding/json"

    "github.com/union-cni/pkg/client"
)

var (
    ErrNoSuchItem = errors.New("no such item")
)


const (
    defaultCtrlPort = "dc0"
    defaultDataPort = "ddc0"
    netInfoKey = "network_info"
)

type NetworkInfo struct {
    Crediential string `json:"credential"`
    Group       string `json:"group"`
    CtrlPort    string `json:"control_port"`
    DataPort    string `json:"data_port"`
    ExternalPort []struct {
        HostPort string `json:"host_port"`
        ContainerPort string `json:"container_port"`
        MapType string `json:"type"`
    } `json:"external_port"`
}

func GetNetInfo(host string, port string, k8sNamespace string, podName string) (*NetworkInfo, error) {
    k8scli := client.CreateInsecureClient(host, port)
    pod, err := k8scli.GetPod(k8sNamespace, podName)
    if err != nil {
         fmt.Fprintf(os.Stderr, "[UNION CNI] failed to get pod: %v\r\n", err)
         return nil, err
    }

    rawData, ok := pod.Annotations[netInfoKey]
    if !ok {
        fmt.Fprintf(os.Stderr, "[UNION CNI] no annotation: network_info\r\n")
        return nil, nil
    }

    netInfo := &NetworkInfo{}
    err = json.Unmarshal([]byte(rawData), netInfo)
    if err != nil {
        fmt.Fprintf(os.Stderr, "[UNION CNI] failed to convert raw %s to json: %v\r\n", err)
        return nil, err
    }

    // field checks
    if len(netInfo.Crediential) == 0 {
        fmt.Fprintf(os.Stderr, "[UNION CNI] No such item: crediential\r\n")
        return nil, ErrNoSuchItem
    }
 
    if len(netInfo.Group) == 0 {
        fmt.Fprintf(os.Stderr, "[UNION CNI] No such item: group\r\n")
        return nil, ErrNoSuchItem
    }
    return netInfo, nil
}

func (netInfo *NetworkInfo)GetDataBridgeName() string {
    return fmt.Sprintf("%s-%s-data", netInfo.Crediential, netInfo.Group)
}

func (netInfo *NetworkInfo)GetCtrlBridgeName() string {
    return fmt.Sprintf("%s-%s-ctrl", netInfo.Crediential, netInfo.Group)
}

func (netInfo *NetworkInfo)GetCtrlPort() string {
    if netInfo.CtrlPort == "" {
        return defaultCtrlPort
    }
    return netInfo.CtrlPort
}

func (netInfo *NetworkInfo)GetDataPort() string {
    if netInfo.DataPort == "" {
        return defaultDataPort
    }
    return netInfo.DataPort
}
