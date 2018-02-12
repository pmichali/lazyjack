package orca

import (
	"fmt"
	"net"

	"github.com/golang/glog"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netlink/nl"
)

func BuildCIDR(subnet string, node, prefix int) string {
	return fmt.Sprintf("%s%d/%d", subnet, node, prefix)
}

func AddAddressToLink(ip, intf string) error {
	link, err := netlink.LinkByName(intf)
	if err != nil {
		return fmt.Errorf("Unable to find interface %q", intf)
	}
	addr, err := netlink.ParseAddr(ip)
	if err != nil {
		return fmt.Errorf("Malformed address %q", ip)
	}
	err = netlink.AddrReplace(link, addr)
	if err != nil {
		return fmt.Errorf("Unable to add ip %q to interface %q", ip, intf)
	}
	glog.V(1).Infof("Added ip %q to interface %q", ip, intf)
	return nil
}

func AddressExistsOnLink(addr *netlink.Addr, link netlink.Link) bool {
	addrs, err := netlink.AddrList(link, nl.FAMILY_ALL)
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

func RemoveAddressFromLink(ip, intf string) error {
	link, err := netlink.LinkByName(intf)
	if err != nil {
		return fmt.Errorf("Unable to find interface %q", intf)
	}
	addr, err := netlink.ParseAddr(ip)
	if err != nil {
		return fmt.Errorf("Malformed address to delete %q", ip)
	}
	if !AddressExistsOnLink(addr, link) {
		glog.V(1).Infof("Skipping - address %q does not exist on interface %q", ip, intf)
		return nil
	}

	err = netlink.AddrDel(link, addr)
	if err != nil {
		return fmt.Errorf("Unable to delete ip %q from interface %q", ip, intf)
	}
	glog.V(1).Infof("Removed ip %q from interface %q", ip, intf)
	return nil
}

func FindLinkIndexForV4CIDR(v4CIDR string) (int, error) {
	_, cidr, err := net.ParseCIDR(v4CIDR)
	if err != nil {
		return 0, fmt.Errorf("Unable to parse V4 CIDR %q: %s", v4CIDR, err.Error())
	}
	links, _ := netlink.LinkList()
	for _, link := range links {
		addrs, _ := netlink.AddrList(link, nl.FAMILY_V4)
		for _, addr := range addrs {
			if cidr.Contains(addr.IP) {
				glog.V(4).Infof("Using interface %s (%d) for CIDR %q", link.Attrs().Name, link.Attrs().Index, v4CIDR)
				return link.Attrs().Index, nil
			}
		}
	}
	return 0, fmt.Errorf("Unable to find interface for V4 CIDR %q", v4CIDR)
}

func BuildV4Route(destStr, gwStr string, index int) (*netlink.Route, error) {
	_, cidr, err := net.ParseCIDR(destStr)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse destination CIDR %q: %s", destStr, err.Error())
	}
	gw := net.ParseIP(gwStr)
	if gw == nil {
		return nil, fmt.Errorf("Unable to parse gateway IP %q", gwStr)
	}
	route := &netlink.Route{Dst: cidr, Gw: gw, LinkIndex: index}
	return route, nil
}

func AddLocalV4RouteToNAT64Server(dest, gw, v4supportCIDR string) error {
	index, err := FindLinkIndexForV4CIDR(v4supportCIDR)
	if err != nil {
		return err
	}
	route, err := BuildV4Route(dest, gw, index)
	if err != nil {
		return err
	}
	return netlink.RouteAdd(route)
}

func RemoveLocalIPv4RouteFromNAT64(dest, gw, v4supportCIDR string) error {
	index, err := FindLinkIndexForV4CIDR(v4supportCIDR)
	if err != nil {
		return err
	}
	route, err := BuildV4Route(dest, gw, index)
	if err != nil {
		return err
	}
	return netlink.RouteDel(route)
	return nil
}
