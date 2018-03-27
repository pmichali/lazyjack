package lazyjack

import (
	"net"

	"github.com/vishvananda/netlink"
)

type NetLink struct {
	h *netlink.Handle
}

// Wrappers for the netlink library...

func (n NetLink) AddrDel(link netlink.Link, addr *netlink.Addr) error {
	return n.h.AddrDel(link, addr)
}

func (n NetLink) AddrList(link netlink.Link, family int) ([]netlink.Addr, error) {
	return netlink.AddrList(link, family)
}

func (n NetLink) AddrReplace(link netlink.Link, addr *netlink.Addr) error {
	return n.h.AddrReplace(link, addr)
}

func (n NetLink) LinkByName(name string) (netlink.Link, error) {
	return n.h.LinkByName(name)
}

func (n NetLink) LinkList() ([]netlink.Link, error) {
	return n.h.LinkList()
}

func (n NetLink) ParseAddr(s string) (*netlink.Addr, error) {
	return netlink.ParseAddr(s)
}

func (n NetLink) ParseIPNet(s string) (*net.IPNet, error) {
	return netlink.ParseIPNet(s)
}

func (n NetLink) RouteAdd(route *netlink.Route) error {
	return n.h.RouteAdd(route)
}

func (n NetLink) RouteDel(route *netlink.Route) error {
	return n.h.RouteDel(route)
}

func (n NetLink) LinkSetDown(link netlink.Link) error {
	return n.h.LinkSetDown(link)
}

func (n NetLink) LinkDel(link netlink.Link) error {
	return n.h.LinkDel(link)
}
