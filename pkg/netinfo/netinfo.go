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
    netInfoKey = "network_info"
)

type ExternalInfo struct {
    HostPort string         `json:"host_port"`
    ContainerPort string    `json:"container_port"`
    Type string             `json:"type"`
    Mode string		    `json:"mode"`
    IP string               `json:"ipaddr"`
}

type NetworkInfo struct {
    Crediential string `json:"credential"`
    Group       string `json:"group"`
    DeviceID    string `json:"deviceid"`
    SystemChan  map[string]string  `json:"system_channels"`
    ExternalPort []ExternalInfo `json:"external_ports"`
}

func (netInfo *NetworkInfo)GetExternalPorts() ([]ExternalInfo) {
    externals := netInfo.ExternalPort
    if len(externals) != 0 {
       return externals
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
        return nil, fmt.Errorf("no annotation: network_info")
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

func (netInfo *NetworkInfo)GetSystemChannels() (map[string]string) {
    return netInfo.SystemChan
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
