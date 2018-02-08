package orca

import (
	"fmt"

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
