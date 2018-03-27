package lazyjack

import (
	"net"

	"github.com/vishvananda/netlink"
)

// NetLinkAPI interface representing low level API to netlink package.
// Used to allow mocking for testing.
type NetLinkAPI interface {
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
