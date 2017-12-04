package bridge

import (
    "fmt"
    "os"
    "syscall"

    "github.com/containernetworking/cni/pkg/types/current"
    "github.com/vishvananda/netlink"
)

const defaultMTU = 1400
const defaultPromiscMode = true

type Bridge struct {
    Data *netlink.Bridge
    Name string
}

func (br *Bridge)BridgeInterface() (*current.Interface) {
    return &current.Interface {
        Name: br.Data.Attrs().Name,
        Mac: br.Data.Attrs().HardwareAddr.String(),
    }
}

func BridgeByName(name string) (*netlink.Bridge, error) {
    link, err := netlink.LinkByName(name)
    if err != nil {
        return nil, fmt.Errorf("could not lookup %q: %v", name, err)
    }
    
    br, ok := link.(*netlink.Bridge)
    if !ok {
        return nil, fmt.Errorf("%q already exists but is not a bridge", name)
    }
    
    return br, nil
}

func (br *Bridge)AddLink(link netlink.Link) error {
    if err := netlink.LinkSetMaster(link, br.Data); err != nil {
        fmt.Fprintf(os.Stderr, "[UNION CNI] failed to add link %q to bridge %q: %v\r\n", link, br.Name, err)
        return err
    }
    return nil
}

func ensureBridge(name string, mtu int, promiscMode bool) (*netlink.Bridge, error) {
    br := &netlink.Bridge{
        LinkAttrs: netlink.LinkAttrs{
            Name: name,
            MTU: mtu,
            // Let kernel use default txqueuelen; leaving it unset
            // means 0, and a zero-length TX queue messes up FIFO
            // traffic shapers which use TX queue length as the 
            // default packet limit
            TxQLen: -1,   
        },
    }

    err := netlink.LinkAdd(br)
    if err != nil && err != syscall.EEXIST {
        return nil, fmt.Errorf("could not add %q: %v", name, err)
    }

    if promiscMode {
        if err := netlink.SetPromiscOn(br); err != nil {
            return nil, fmt.Errorf("could not set promiscuous mode on %q: %v", name, err)
        }
    }

    // Re-fetch link to read all attributes and if it already exisited,
    // ensure it's really a bridge with similar configuration
    br, err = BridgeByName(name)
    if err != nil {
        return nil, err
    }
   
    if err := netlink.LinkSetUp(br); err != nil {
        return nil, err
    }

    return br, nil
}

func CreateBridge(name string) (*Bridge, error) {
    br, err := ensureBridge(name, defaultMTU, defaultPromiscMode)
    if err != nil {
        return nil, fmt.Errorf("failed to create bridge %q: %v", name, err)
    }

    bridge := &Bridge{
        Data: br,
        Name: name,
    }

    return bridge, nil
}
