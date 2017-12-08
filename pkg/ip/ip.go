package ip

import (
    "fmt"
    "os"

    "github.com/containernetworking/plugins/pkg/ns"
    "github.com/vishvananda/netlink"
)

func AddrAddInNS(linkName string, ipaddr string, nspath string) error {
    addr, _ := netlink.ParseAddr(ipaddr)

    netNS,err := ns.GetNS(nspath)
    if err != nil {
        return err
    }

    err = netNS.Do(func (_ ns.NetNS) error {
            l, err := netlink.LinkByName(linkName)
            if err != nil {
                fmt.Fprintf(os.Stderr, "[UNION CNI] failed to lookup %q in %q: %v\r\n", linkName, nspath, err)
                return err
            }

            return netlink.AddrAdd(l, addr)
    })

    netNS.Close()

    return err
}
