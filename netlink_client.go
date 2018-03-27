package lazyjack

import (
	"net"

	"github.com/vishvananda/netlink"
)

// NetLink defines the structure for netlink implementation of networking
type NetLink struct {
	h *netlink.Handle
}

// Wrappers for the netlink library...

// AddrDel deletes an address from a link
func (n NetLink) AddrDel(link netlink.Link, addr *netlink.Addr) error {
	return n.h.AddrDel(link, addr)
}

// AddrList lists IPs on a link
func (n NetLink) AddrList(link netlink.Link, family int) ([]netlink.Addr, error) {
	return netlink.AddrList(link, family)
}

// AddrReplace replaces IP address on a link
func (n NetLink) AddrReplace(link netlink.Link, addr *netlink.Addr) error {
	return n.h.AddrReplace(link, addr)
}

// LinkByName finds a link by name
func (n NetLink) LinkByName(name string) (netlink.Link, error) {
	return n.h.LinkByName(name)
}

// LinkList lists all links on system
func (n NetLink) LinkList() ([]netlink.Link, error) {
	return n.h.LinkList()
}

// ParseAddr parses the string as an IP address
func (n NetLink) ParseAddr(s string) (*netlink.Addr, error) {
	return netlink.ParseAddr(s)
}

// ParseIPNet parses the string as an IPNet object.
func (n NetLink) ParseIPNet(s string) (*net.IPNet, error) {
	return netlink.ParseIPNet(s)
}

// RouteAdd adds a route
func (n NetLink) RouteAdd(route *netlink.Route) error {
	return n.h.RouteAdd(route)
}

// RouteDel deletes a route
func (n NetLink) RouteDel(route *netlink.Route) error {
	return n.h.RouteDel(route)
}

// LinkSetDown brings down an interface
func (n NetLink) LinkSetDown(link netlink.Link) error {
	return n.h.LinkSetDown(link)
}

// LinkDel deletes an interface
func (n NetLink) LinkDel(link netlink.Link) error {
	return n.h.LinkDel(link)
}
