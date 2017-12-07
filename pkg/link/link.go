package link

import (
    "github.com/vishvananda/netlink"
    "github.com/containernetworking/cni/pkg/types/current"
)

func Interface(l netlink.Link, sandbox string) *current.Interface {
    if sandbox != "" {
        return &current.Interface{
            Name: l.Attrs().Name,
            Mac: l.Attrs().HardwareAddr.String(),
            Sandbox: sandbox,
        }
    } else {
        return &current.Interface{
            Name: l.Attrs().Name,
            Mac: l.Attrs().HardwareAddr.String(),
        }
    }
}
