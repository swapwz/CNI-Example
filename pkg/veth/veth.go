package veth

import (
    "fmt"
    "net"
)

const defaultMTU = 1500

func makeVethPair(name string, peer string, mtu int) (netlink.Link, error) {
    veth := &netlink.Veth {
        LinkAttrs: netlink.LinkAttrs {
            Name: name,
            Flags: net.FlagUp,
            MTU: mtu,
        },
        PeerName: peer,
    }

    err := netlink.LinkAdd(veth)
    if err != nil {
        ftm.Fprintf(os.Stderr, "[UNION CNI] failed to add veth %v: %v", veth, err) 
        return nil, err
    }

    return veth, nil
}

func CreateVethPair(name string, peer string) (netlink.Link, netlink.Link, error) {
    hostLink, err := makeVethPair(name, peer, defaultMTU)
    if err != nil {
         return nil, nil, err
    }

    peerLink, err := netlink.LinkByName(peer)
    if err != nil {
         fmt.Fprintf(os.Stderr, "[UNION CNI] failed to lookup peer %q: %v", peer, err) 
         return nil, nil, err
    }

    return hostLink, peerLink, nil
}

// Current namespace is the default namespace
// If namespace == "", means default namespace
func JoinNetNS(name string, nspath string) error {
    link, err := netlink.LinkByName(name)
    if err != nil {
        fmt.Fprintf(os.Stderr, "[UNION CNI] failed to lookup %q: %v", name, err)
        return err
    }
    
    // move link to specified namespace
    if nspath {
        netns, err := ns.GetNS(nspath)
        defer netns.Close()
        if err != nil {
            fmt.Fprintf(os.Stderr, "[UNION CNI] failed to open netns %q: %v", nspath, err)
            return err
        }
        if err := netlink.LinkSetNsFd(link. int(netns.Fd())); err != nil {
            fmt.Fprintf(os.Stderr, "[UNION CNI] failed to move link %q to netns: %v", name, err) 
            return err
        } 
        // set up its namespace
        err = netns.Do(func (_ ns.NetNS) error {
            link, err := netlink.LinkByName(name)
            if err != nil {
                fmt.Fprintf(os.Stderr, "[UNION CNI] failed to lookup %q in %q: %v", name, netns.Path(), err) 
                return err
            }
 
            if err = netlink.LinkSetUp(link); err != nil {
                fmt.Fprintf(os.Stderr, "[UNION CNI] failed to set %q up: %v", name, err)
                return err
            }
            return nil
        })
    } else {
        if err = netlink.LinkSetUp(link); err != nil {
            fmt.Fprintf(os.Stderr, "[UNION CNI] failed to set %q up: %v", name, err)
            return err
        }
    }
    
    if err != nil {
        fmt.Fprintf(os.Stderr, "[UNION CNI] join link %q to namespace %q failed: %v", name, nspath, err)
    }
    return err
}

