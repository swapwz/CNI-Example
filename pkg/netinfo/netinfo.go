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

type ExternalInfo struct {
    HostPort string
    ContainerPort string
    Type string
}

type NetworkInfo struct {
    Crediential string `json:"credential"`
    Group       string `json:"group"`
    DeviceID    string `json:"deviceid"`
    CtrlPort    string `json:"control_port"`
    DataPort    string `json:"data_port"`
    ExternalPort []struct {
        HostPort string `json:"host_port"`
        ContainerPort string `json:"container_port"`
        MapType string `json:"type"`
    } `json:"external_port"`
}

func (netInfo *NetworkInfo)GetExternalPorts() ([]ExternalInfo) {
    externals := netInfo.ExternalPort
    if len(externals) != 0 {
        extInfo := make([]ExternalInfo, len(externals))
        for index, ext := range externals {
           fmt.Fprintf(os.Stderr, "[UNION CNI] external %v\r\n", ext)
           extInfo[index].HostPort = ext.HostPort
           extInfo[index].ContainerPort = ext.ContainerPort
           extInfo[index].Type = ext.MapType
       }
       return extInfo
    } else {
       return nil
    }
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

func (netInfo *NetworkInfo)GetDeviceID() string {
    return netInfo.DeviceID
}

func (netInfo *NetworkInfo)GetCred() string {
    return netInfo.Crediential
}

func (netInfo *NetworkInfo)GetGroup() string {
    return netInfo.Group
}
