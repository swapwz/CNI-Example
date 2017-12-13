package link

import (
    "fmt"

    "github.com/containernetworking/plugins/pkg/ns"
    "github.com/vishvananda/netlink"
)

func modeFromString(s string) (netlink.MacvlanMode, error) {
    switch s {
    case "bridge":
            return netlink.MACVLAN_MODE_BRIDGE, nil
    case "private":
            return netlink.MACVLAN_MODE_PRIVATE, nil
    case "vepa":
            return netlink.MACVLAN_MODE_VEPA, nil
    case "passthru":
            return netlink.MACVLAN_MODE_PASSTHRU, nil
    default:
            return 0, fmt.Errorf("unknown macvlan mode: %q", s)
    }
}


func CreateMacvlanInNS(hostPort string, conPort string, mode string, nspath string) (cLink netlink.Link, err error) {
    set_mode := netlink.MACVLAN_MODE_VEPA
    if mode != "" {
        set_mode,err = modeFromString(mode) 
    }     

    if err != nil {
        return
    }

    hLink, err := netlink.LinkByName(hostPort)
    if err != nil {
        return nil, fmt.Errorf("failed to lookup physical port %q: %v", hostPort, err)
    }

    tmpName, err := getRandomName()  
    if err != nil {
        return nil, err
    }

    netns,_ := ns.GetNS(nspath)
    defer netns.Close()

    tmpLink := &netlink.Macvlan{
        LinkAttrs: netlink.LinkAttrs{
            MTU:         defaultMtu,
            Name:        tmpName,
            ParentIndex: hLink.Attrs().Index,
            Namespace:   netlink.NsFd(int(netns.Fd())),
        },
        Mode: set_mode,
    }

    if err = netlink.LinkAdd(tmpLink); err != nil {
         return nil, fmt.Errorf("failed to create macvlan: %v", err)
    }

    // fixed the name in namespace
    err = netns.Do(func(_ ns.NetNS) error {
        // rename
        tLink, err := netlink.LinkByName(tmpName)
        if err == nil {
            err = netlink.LinkSetName(tLink, conPort)
        }
        if err != nil {
            _ = netlink.LinkDel(tmpLink)
            return fmt.Errorf("failed to rename macvlan to %q: %v", conPort, err)
        }

        cLink, err = netlink.LinkByName(conPort)
        netlink.LinkSetUp(cLink)
        netlink.SetPromiscOn(cLink)
        return err
    }) 
   
    return 
}
