package lazyjack

import (
	"net"

	"github.com/vishvananda/netlink"
)

// Interface representing external package
type network interface {
	AddrDel(netlink.Link, *netlink.Addr) error
	AddrList(netlink.Link, int) ([]netlink.Addr, error)
	AddrReplace(netlink.Link, *netlink.Addr) error
	LinkByName(name string) (netlink.Link, error)
	LinkList() ([]netlink.Link, error)
	ParseAddr(string) (*netlink.Addr, error)
	ParseIPNet(s string) (*net.IPNet, error)
	RouteAdd(route *netlink.Route) error
	RouteDel(route *netlink.Route) error
	LinkSetDown(link netlink.Link) error
	LinkDel(link netlink.Link) error
}

type NetManager struct {
	Mgr network
}

type RealImpl struct {
	h *netlink.Handle
}

// Wrappers for the netlink library...

func (r *RealImpl) AddrDel(link netlink.Link, addr *netlink.Addr) error {
	return r.h.AddrDel(link, addr)
}

func (r *RealImpl) AddrList(link netlink.Link, family int) ([]netlink.Addr, error) {
	return netlink.AddrList(link, family)
}

func (r *RealImpl) AddrReplace(link netlink.Link, addr *netlink.Addr) error {
	return r.h.AddrReplace(link, addr)
}

func (r *RealImpl) LinkByName(name string) (netlink.Link, error) {
	return r.h.LinkByName(name)
}

func (r *RealImpl) LinkList() ([]netlink.Link, error) {
	return r.h.LinkList()
}

func (r *RealImpl) ParseAddr(s string) (*netlink.Addr, error) {
	return netlink.ParseAddr(s)
}

func (r *RealImpl) ParseIPNet(s string) (*net.IPNet, error) {
	return netlink.ParseIPNet(s)
}

func (r *RealImpl) RouteAdd(route *netlink.Route) error {
	return r.h.RouteAdd(route)
}

func (r *RealImpl) RouteDel(route *netlink.Route) error {
	return r.h.RouteDel(route)
}

func (r *RealImpl) LinkSetDown(link netlink.Link) error {
	return r.h.LinkSetDown(link)
}

func (r *RealImpl) LinkDel(link netlink.Link) error {
	return r.h.LinkDel(link)
}
