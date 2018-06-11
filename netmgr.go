package lazyjack

import (
	"fmt"
	"net"

	"github.com/golang/glog"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netlink/nl"
)

// NetMgr defines the structure for a netlink implementation for networking.
type NetMgr struct {
	Server NetLinkAPI
}

// AddAddressToLink method adds an IP address to a link, replacing the
// existing address, if any.
func (n NetMgr) AddAddressToLink(ip, intf string) error {
	link, err := n.Server.LinkByName(intf)
	if err != nil {
		return fmt.Errorf("unable to find interface %q", intf)
	}
	addr, err := n.Server.ParseAddr(ip)
	if err != nil {
		return fmt.Errorf("malformed address %q", ip)
	}
	err = n.Server.AddrReplace(link, addr)
	if err != nil {
		return fmt.Errorf("unable to add ip %q to interface %q", ip, intf)
	}
	glog.V(1).Infof("Added ip %q to interface %q", ip, intf)
	return nil
}

// AddressExistsOnLink checks to see if the address to be deleted, is on
// the link. If we are unable to obtain link info, we'll assume address
// is not there, and will later skip trying to remove it.
func (n NetMgr) AddressExistsOnLink(addr *netlink.Addr, link netlink.Link) bool {
	addrs, err := n.Server.AddrList(link, nl.FAMILY_ALL)
	if err != nil {
		return false // Will assume it exists
	}
	for _, a := range addrs {
		if addr.Equal(a) {
			return true
		}
	}
	return false
}

// RemoveAddressFromLink method removes an IP addres from an interface.
func (n NetMgr) RemoveAddressFromLink(ip, intf string) error {
	link, err := n.Server.LinkByName(intf)
	if err != nil {
		return fmt.Errorf("unable to find interface %q", intf)
	}
	addr, err := n.Server.ParseAddr(ip)
	if err != nil {
		return fmt.Errorf("malformed address to delete %q", ip)
	}
	if !n.AddressExistsOnLink(addr, link) {
		return fmt.Errorf("skipping - address %q does not exist on interface %q", ip, intf)
	}

	err = n.Server.AddrDel(link, addr)
	if err != nil {
		return fmt.Errorf("unable to delete ip %q from interface %q", ip, intf)
	}
	glog.V(1).Infof("Removed ip %q from interface %q", ip, intf)
	return nil
}

// FindLinkIndexForCIDR method obtains the index of the interface that
// contains the CIDR.
func (n NetMgr) FindLinkIndexForCIDR(cidr string) (int, error) {
	c, err := n.Server.ParseIPNet(cidr)
	if err != nil {
		return 0, err
	}
	links, _ := n.Server.LinkList()
	if len(links) == 0 {
		return 0, fmt.Errorf("no links on system")
	}
	for _, link := range links {
		addrs, _ := n.Server.AddrList(link, nl.FAMILY_V4)
		for _, addr := range addrs {
			if c.Contains(addr.IP) {
				glog.V(4).Infof("Using interface %s (%d) for CIDR %q", link.Attrs().Name, link.Attrs().Index, cidr)
				return link.Attrs().Index, nil
			}
		}
	}
	return 0, fmt.Errorf("unable to find interface for CIDR %q", cidr)
}

// BuildRoute creates a route to the destination, using the provided
// gateway.
func BuildRoute(destStr, gwStr string, index int) (*netlink.Route, error) {
	_, cidr, err := net.ParseCIDR(destStr)
	if err != nil {
		return nil, fmt.Errorf("unable to parse destination CIDR %q: %v", destStr, err)
	}
	gw := net.ParseIP(gwStr)
	if gw == nil {
		return nil, fmt.Errorf("unable to parse gateway IP %q", gwStr)
	}
	route := &netlink.Route{Dst: cidr, Gw: gw, LinkIndex: index}
	return route, nil
}

// AddRouteUsingSupportNetInterface method adds a route to the destination
// using the gateway and support network CIDR.
func (n NetMgr) AddRouteUsingSupportNetInterface(dest, gw, supportNetCIDR string) error {
	glog.V(4).Infof("Adding route for %s via %s using CIDR %s for link determination", dest, gw, supportNetCIDR)
	index, err := n.FindLinkIndexForCIDR(supportNetCIDR)
	if err != nil {
		return err
	}
	route, err := BuildRoute(dest, gw, index)
	if err != nil {
		return err
	}
	return n.Server.RouteAdd(route)
}

// DeleteRouteUsingSupportNetInterface method removes the route to the
// destination that uses the provided CIDR.
func (n NetMgr) DeleteRouteUsingSupportNetInterface(dest, gw, supportNetCIDR string) error {
	glog.V(4).Infof("Deleting route for %s via %s using CIDR %s for link determination", dest, gw, supportNetCIDR)
	index, err := n.FindLinkIndexForCIDR(supportNetCIDR)
	if err != nil {
		return err
	}
	route, err := BuildRoute(dest, gw, index)
	if err != nil {
		return err
	}
	return n.Server.RouteDel(route)
}

// AddRouteUsingInterfaceName method adds a route to the destination,
// using the local interface.
func (n NetMgr) AddRouteUsingInterfaceName(dest, gw, intf string) error {
	glog.V(4).Infof("Adding route for %s via %s using interface %s", dest, gw, intf)
	link, err := n.Server.LinkByName(intf)
	if err != nil {
		return fmt.Errorf("unable to find interface %q", intf)
	}
	index := link.Attrs().Index
	route, err := BuildRoute(dest, gw, index)
	if err != nil {
		return err
	}
	return n.Server.RouteAdd(route)
}

// DeleteRouteUsingInterfaceName method removes the route to the destination
// that uses the local interface.
func (n NetMgr) DeleteRouteUsingInterfaceName(dest, gw, intf string) error {
	glog.V(4).Infof("Deleting route for %s via %s using interface %s", dest, gw, intf)
	link, err := n.Server.LinkByName(intf)
	if err != nil {
		return fmt.Errorf("skipping - Unable to find interface %q to delete route", intf)
	}
	index := link.Attrs().Index
	route, err := BuildRoute(dest, gw, index)
	if err != nil {
		return err
	}
	return n.Server.RouteDel(route)
}

// BringLinkDown method shuts down the link specified.
func (n NetMgr) BringLinkDown(name string) error {
	glog.V(4).Infof("Bringing down interface %q", name)
	link, err := n.Server.LinkByName(name)
	if err != nil {
		return fmt.Errorf("unable to find interface %q", name)
	}
	err = n.Server.LinkSetDown(link)
	if err != nil {
		return fmt.Errorf("unable to shut down interface %q", name)
	}
	glog.V(1).Infof("Interface %q brought down", name)
	return nil
}

// DeleteLink method deletes the link specified.
func (n NetMgr) DeleteLink(name string) error {
	glog.V(4).Infof("Deleting interface %q", name)
	link, err := n.Server.LinkByName(name)
	if err != nil {
		return fmt.Errorf("unable to find interface %q", name)
	}
	err = n.Server.LinkDel(link)
	if err != nil {
		return fmt.Errorf("unable to delete interface %q", name)
	}
	glog.V(1).Infof("Deleted interface %q", name)
	return nil
}

// SetLinkMTU method sets the MTU on the link.
func (n NetMgr) SetLinkMTU(name string, mtu int) error {
	glog.V(4).Infof("Setting MTU to %d on interface %q", mtu, name)
	link, err := n.Server.LinkByName(name)
	if err != nil {
		return fmt.Errorf("unable to find interface %q", name)
	}
	err = n.Server.LinkSetMTU(link, mtu)
	if err != nil {
		return fmt.Errorf("unable to set MTU on interface %q", name)
	}
	glog.V(1).Infof("Interface %q MTU set", name)
	return nil
}

// RemoveBridge method removes the specified bridge.
func (n NetMgr) RemoveBridge(name string) error {
	glog.V(1).Infof("Removing bridge %q", name)
	err := n.BringLinkDown(name)
	if err == nil {
		glog.Infof("Brought link %q down", name)
	}
	// Even if err, will try to delete bridge
	err2 := n.DeleteLink(name)
	if err2 == nil {
		glog.Infof("Removed bridge %q", name)
	}
	if err == nil {
		return err2
	} else if err2 == nil {
		return err
	}
	return fmt.Errorf("unable to bring link down (%v), nor remove link (%v)", err, err2)
}
