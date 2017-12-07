package main

import (
    "os"
    "fmt"
    "encoding/json"
)

type ExternalInfo struct {
    HostPort string         `json:"host_port"`
    ContainerPort string    `json:"container_port"`
    Type string             `json:"type"`
} 

type NetworkInfo struct {
    Crediential string `json:"credential"`
    Group       string `json:"group"`
    DeviceID    string `json:"deviceid"`
    SystemChan  map[string]string  `json:"system_channels"`
    ExternalPort []ExternalInfo `json:"external_ports"`
}

func main() {
    f,err := os.Open("test.json") 
    if err != nil {
       fmt.Printf("%v\r\n")
       return
    }
    rawData := make([]byte, 512)
    cnt,_ := f.Read(rawData)
    netInfo := NetworkInfo{}
    err = json.Unmarshal(rawData[:cnt], &netInfo)
    fmt.Printf("Test: %v: %v\r\n", netInfo, err)
}
