package lazyjack

import (
	"fmt"
)

// Networker interface describes the API for networking operations
type Networker interface {
	AddAddressToLink(ip, intf string) error
	RemoveAddressFromLink(ip, intf string) error
	AddRouteUsingSupportNetInterface(dest, gw, supportNetCIDR string) error
	DeleteRouteUsingSupportNetInterface(dest, gw, supportNetCIDR string) error
	AddRouteUsingInterfaceName(dest, gw, intf string) error
	DeleteRouteUsingInterfaceName(dest, gw, intf string) error
	BringLinkDown(name string) error
	DeleteLink(name string) error
	RemoveBridge(name string) error
	SetLinkMTU(name string, mtu int) error
}

// BuildNodeCIDR helper constructs a node CIDR. The network portion
// of the CIDR (the prefix), has the node added as the last part of
// the final address. For example, fd00:20::/64 -> fd00:20::3/64
func BuildNodeCIDR(info NetInfo, node int) string {
	if info.Mode == IPv6NetMode {
		return fmt.Sprintf("%s%x/%d", info.Prefix, node, info.Size)
	} else { // IPv4
		return fmt.Sprintf("%s%d/%d", info.Prefix, node, info.Size)
	}
}

// BuildGWIP helper constructs a gateway IP using the provided interface.
func BuildGWIP(prefix string, intfPart int) string {
	return fmt.Sprintf("%s%x", prefix, intfPart)
}
