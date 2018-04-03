package lazyjack

import "fmt"

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
}

// BuildNodeCIDR helper constructs a node CIDR.
func BuildNodeCIDR(prefix string, node, mask int) string {
	return fmt.Sprintf("%s%d/%d", prefix, node, mask)
}

// BuildDestCIDR helper constructs an IPv6 CIDR.
func BuildDestCIDR(prefix string, node, size int) string {
	return fmt.Sprintf("%s%d::/%d", prefix, node, size)
}

// BuildGWIP helper constructs a gateway IP using the provided interface.
func BuildGWIP(prefix string, intfPart int) string {
	return fmt.Sprintf("%s%d", prefix, intfPart)
}
