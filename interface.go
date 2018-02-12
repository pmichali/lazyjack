package orca

import (
	"fmt"
	"net"

	"github.com/golang/glog"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netlink/nl"
)

func BuildNodeCIDR(subnet string, node, prefix int) string {
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

func FindLinkIndexForCIDR(cidr string) (int, error) {
	_, c, err := net.ParseCIDR(cidr)
	if err != nil {
		return 0, fmt.Errorf("Unable to parse CIDR %q: %s", cidr, err.Error())
	}
	links, _ := netlink.LinkList()
	for _, link := range links {
		addrs, _ := netlink.AddrList(link, nl.FAMILY_V4)
		for _, addr := range addrs {
			if c.Contains(addr.IP) {
				glog.V(4).Infof("Using interface %s (%d) for CIDR %q", link.Attrs().Name, link.Attrs().Index, cidr)
				return link.Attrs().Index, nil
			}
		}
	}
	return 0, fmt.Errorf("Unable to find interface for CIDR %q", cidr)
}

func BuildRoute(destStr, gwStr string, index int) (*netlink.Route, error) {
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

func AddLocalRouteToNAT64Server(dest, gw, supportNetCIDR string) error {
	index, err := FindLinkIndexForCIDR(supportNetCIDR)
	if err != nil {
		return err
	}
	route, err := BuildRoute(dest, gw, index)
	if err != nil {
		return err
	}
	return netlink.RouteAdd(route)
}

func RemoveLocalRouteFromNAT64(dest, gw, supportNetCIDR string) error {
	index, err := FindLinkIndexForCIDR(supportNetCIDR)
	if err != nil {
		return err
	}
	route, err := BuildRoute(dest, gw, index)
	if err != nil {
		return err
	}
	return netlink.RouteDel(route)
}

func BuildDestCIDR(prefix string, node, size int) string {
	return fmt.Sprintf("%s:%d::/%d", prefix, node, size)
}

func BuildGWIP(prefix string, intfPart int) string {
	return fmt.Sprintf("%s%d", prefix, intfPart)
}

func AddRouteForPodNetwork(dest, gw, intf string, node int) error {
	link, err := netlink.LinkByName(intf)
	if err != nil {
		return fmt.Errorf("Unable to find interface %q", intf)
	}
	index := link.Attrs().Index
	route, err := BuildRoute(dest, gw, index)
	if err != nil {
		return err
	}
	return netlink.RouteAdd(route)
}

func DeleteRouteForPodNetwork(dest, gw, intf string, node int) error {
	link, err := netlink.LinkByName(intf)
	if err != nil {
		return fmt.Errorf("Unable to find interface %q", intf)
	}
	index := link.Attrs().Index
	route, err := BuildRoute(dest, gw, index)
	if err != nil {
		return err
	}
	return netlink.RouteDel(route)
}
