package veth

import (
    "fmt"
    "os"
    "net"
    "errors"
    "crypto/rand"
   
    "github.com/containernetworking/cni/pkg/types/current"
    "github.com/containernetworking/plugins/pkg/ns"
    "github.com/vishvananda/netlink"
)

const defaultMTU = 1400
const defaultPrefix = "sim"

var (
    ErrLinkNotFound = errors.New("link not found")
)

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
        fmt.Fprintf(os.Stderr, "[UNION CNI] failed to add veth %v: %v\r\n", veth, err) 
        return nil, err
    }

    return veth, nil
}

func getRandomName() (string, error) {
    entropy := make([]byte, 4)
    _, err := rand.Reader.Read(entropy)
    if err != nil {
        return "", fmt.Errorf("failed to generate random veth name: %v", err)
    }

    return fmt.Sprintf("sim%x", entropy), nil
}

func peerExists(name string) bool {
    if _, err := netlink.LinkByName(name); err != nil {
        return false
    }
    return true
}

func CreateVethPairRandom(name string) (netlink.Link, netlink.Link, error) {
    for i := 0; i < 10; i++ {
        peer, err := getRandomName()
        if err != nil {
            break
        }
        hostLink, peerLink, err := CreateVethPair(name, peer)
        switch {
        case err == nil:
            return hostLink, peerLink, nil
        case os.IsExist(err):
            if peerExists(peer) {
                 continue
            }
            err = fmt.Errorf("container veth name provided (%v) already exists", name)
            return nil, nil, err
        default:
            err = fmt.Errorf("failed to make veth pair: %v", err)
            return nil, nil, err
        }
    }

    err :=  fmt.Errorf("[UNION CNI] failed to find a unique veth name")
    return nil, nil, err
}

func CreateVethPair(name string, peer string) (netlink.Link, netlink.Link, error) {
    hostLink, err := makeVethPair(name, peer, defaultMTU)
    if err != nil {
         return nil, nil, err
    }

    peerLink, err := netlink.LinkByName(peer)
    if err != nil {
         fmt.Fprintf(os.Stderr, "[UNION CNI] failed to lookup peer %q: %v\r\n", peer, err) 
         return nil, nil, err
    }

    return hostLink, peerLink, nil
}

func DelLink(name string) error {
    link, err := netlink.LinkByName(name)
    if err != nil {
        if err.Error() == "Link not found" {
             return ErrLinkNotFound
        }
        return fmt.Errorf("failed to lookup %q: %v", name, err)
    }
   
    if err = netlink.LinkDel(link); err != nil {
        return fmt.Errorf("failed to delete %q: %v", name, err)
    }

    return nil
}

func DelLinkInNS(name string, nspath string) error {
    netns, err := ns.GetNS(nspath)
    if err != nil {
        return err
    }
    defer netns.Close()

    return DelLink(name)
}

// Current namespace is the default namespace
// If namespace == "", means default namespace
func JoinNetNS(name string, nspath string) error {
    link, err := netlink.LinkByName(name)
    if err != nil {
        fmt.Fprintf(os.Stderr, "[UNION CNI] failed to lookup %q: %v\r\n", name, err)
        return err
    }
    
    // move link to specified namespace
    if nspath != "" {
        netns, err := ns.GetNS(nspath)
        if err != nil {
            fmt.Fprintf(os.Stderr, "[UNION CNI] failed to open netns %q: %v\r\n", nspath, err)
            return err
        }
        defer netns.Close()
        if err := netlink.LinkSetNsFd(link, int(netns.Fd())); err != nil {
            fmt.Fprintf(os.Stderr, "[UNION CNI] failed to move link %q to netns: %v\r\n", name, err) 
            return err
        } 
        // set up its namespace
        err = netns.Do(func (_ ns.NetNS) error {
            link, err := netlink.LinkByName(name)
            if err != nil {
                fmt.Fprintf(os.Stderr, "[UNION CNI] failed to lookup %q in %q: %v\r\n", name, netns.Path(), err) 
                return err
            }
 
            if err = netlink.LinkSetUp(link); err != nil {
                fmt.Fprintf(os.Stderr, "[UNION CNI] failed to set %q up: %v\r\n", name, err)
                return err
            }
            return nil
        })
    } else {
        if err = netlink.LinkSetUp(link); err != nil {
            fmt.Fprintf(os.Stderr, "[UNION CNI] failed to set %q up: %v\r\n", name, err)
            return err
        }
    }
    
    if err != nil {
        fmt.Fprintf(os.Stderr, "[UNION CNI] join link %q to namespace %q failed: %v\r\n", name, nspath, err)
    }
    return err
}

func VethInterface(link netlink.Link, netns ns.NetNS) *current.Interface {
    netIntf := net.Interface {
        Index: link.Attrs().Index,
        MTU: link.Attrs().MTU,
        Name: link.Attrs().Name,
        HardwareAddr: link.Attrs().HardwareAddr,
        Flags: link.Attrs().Flags,
    }

    return &current.Interface{
        Name: netIntf.Name,
        Mac: netIntf.HardwareAddr.String(),
        Sandbox: netns.Path(),
    }
}

