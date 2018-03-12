package lazyjack_test

import (
	"fmt"
	"net"
	"strconv"
	"testing"

	"github.com/pmichali/lazyjack"
	"github.com/vishvananda/netlink"
)

type mockImpl struct {
	simLookupFail    bool
	simParseAddrFail bool
	simDeleteFail    bool
	simReplaceFail   bool
	simAddrListFail  bool
	simLinkListFail  bool
	simRouteAddFail  bool
	simRouteDelFail  bool
}

func (m *mockImpl) AddrDel(link netlink.Link, addr *netlink.Addr) error {
	if m.simDeleteFail {
		return fmt.Errorf("Mock failure to delete address")
	}
	return nil
}

func (m *mockImpl) AddrList(link netlink.Link, family int) ([]netlink.Addr, error) {
	if m.simAddrListFail {
		return []netlink.Addr{}, fmt.Errorf("Mock failure to list addresses")
	}
	// Will use the link index to create dummy addresses per link
	first, _ := netlink.ParseAddr(fmt.Sprintf("2001:db8:%d::2/64", link.Attrs().Index))
	second, _ := netlink.ParseAddr(fmt.Sprintf("2001:db8:%d::3/64", link.Attrs().Index))
	addrList := []netlink.Addr{*first, *second}
	return addrList, nil
}

func (m *mockImpl) AddrReplace(link netlink.Link, addr *netlink.Addr) error {
	if m.simReplaceFail {
		return fmt.Errorf("Mock failure to replace address")
	}
	return nil
}

func (m *mockImpl) LinkByName(name string) (netlink.Link, error) {
	if m.simLookupFail {
		return nil, fmt.Errorf("Mock failure to find link")
	}
	// Calc index based on interface name, using last digit * 10.
	// For example "eth2" -> 2*10 = 20.
	link := &netlink.Device{}
	idx, _ := strconv.Atoi(name[len(name)-1:])
	link.Index = idx * 10
	return link, nil
}

func (m *mockImpl) LinkList() ([]netlink.Link, error) {
	if m.simLinkListFail {
		return []netlink.Link{}, fmt.Errorf("Mock failure to list addresses")
	}
	// Making a dummy list of two entries with indexes 20 and 30. The dummy addresses
	// we create for some tests, will use the index as part of the IP.
	linkA := &netlink.Device{}
	linkA.Index = 20
	linkB := &netlink.Device{}
	linkB.Index = 30
	return []netlink.Link{linkA, linkB}, nil
}

func (m *mockImpl) ParseAddr(s string) (*netlink.Addr, error) {
	if m.simParseAddrFail {
		return nil, fmt.Errorf("Mock failure to parse address")
	}
	return netlink.ParseAddr(s)
}

func (m *mockImpl) ParseIPNet(s string) (*net.IPNet, error) {
	return netlink.ParseIPNet(s)
}

func (m *mockImpl) RouteAdd(route *netlink.Route) error {
	if m.simRouteAddFail {
		return fmt.Errorf("Mock failure adding route")
	}
	return nil
}

func (m *mockImpl) RouteDel(route *netlink.Route) error {
	if m.simRouteDelFail {
		return fmt.Errorf("Mock failure deleting route")
	}
	return nil
}

// START OF TESTS...
func TestBuildNodeCIDR(t *testing.T) {
	actual := lazyjack.BuildNodeCIDR("2001:db8:20::", 2, 64)
	expected := "2001:db8:20::2/64"
	if actual != expected {
		t.Errorf("FAILED: Node CIDR create. Expected %q, got %q", expected, actual)
	}
}

func TestAddAddressToLink(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
	err := nm.AddAddressToLink("2001:db8::10/64", "eth1")
	if err != nil {
		t.Errorf("FAILED: Expected address add to pass: %s", err.Error())
	}
}

func TestFailedLookupForAddAddressToLink(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{simLookupFail: true}}
	err := nm.AddAddressToLink("2001:db8::10", "eth1")
	if err == nil {
		t.Errorf("FAILED: Expected address add to fail")
	}
	expectedErr := "Unable to find interface \"eth1\""
	if err.Error() != expectedErr {
		t.Errorf("FAILED: Expected failure message %q, got %q", expectedErr, err.Error())
	}
}

func TestFailedParseForAddAddressToLink(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{simParseAddrFail: true}}
	err := nm.AddAddressToLink("2001::db8::10/64", "eth1")
	if err == nil {
		t.Errorf("FAILED: Expected address add to fail")
	}
	expectedErr := "Malformed address \"2001::db8::10/64\""
	if err.Error() != expectedErr {
		t.Errorf("FAILED: Expected failure message %q, got %q", expectedErr, err.Error())
	}
}

func TestFailedReplaceForAddAddressToLink(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{simReplaceFail: true}}
	err := nm.AddAddressToLink("2001:db8::10/64", "eth1")
	if err == nil {
		t.Errorf("FAILED: Expected address add to fail")
	}
	expectedErr := "Unable to add ip \"2001:db8::10/64\" to interface \"eth1\""
	if err.Error() != expectedErr {
		t.Errorf("FAILED: Expected failure message %q, got %q", expectedErr, err.Error())
	}
}

func TestBuildDestCIDR(t *testing.T) {
	actual := lazyjack.BuildDestCIDR("2001:db8", 20, 80)
	expected := "2001:db8:20::/80"
	if actual != expected {
		t.Errorf("FAILED: Destination CIDR create. Expected %q, got %q", expected, actual)
	}
}

func TestFailureAddressExistsOnLink(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{simAddrListFail: true}}
	dummyAddr, _ := netlink.ParseAddr("2001:db8:10::2/64")
	dummyLink := &netlink.Device{}
	dummyLink.Index = 10
	exists := nm.AddressExistsOnLink(dummyAddr, dummyLink)
	if exists {
		t.Errorf("FAILED: Expected address to not exist on link")
	}
}

func TestNotFoundAddressExistsOnLink(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
	dummyAddr, _ := netlink.ParseAddr("2001:db8:50::2/64")
	dummyLink := &netlink.Device{}
	dummyLink.Index = 10
	exists := nm.AddressExistsOnLink(dummyAddr, dummyLink)
	if exists {
		t.Errorf("FAILED: Expected address to not exist on link")
	}
}

func TestAddressExistsOnLink(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
	dummyAddr, _ := netlink.ParseAddr("2001:db8:30::2/64")
	dummyLink := &netlink.Device{}
	dummyLink.Index = 30
	exists := nm.AddressExistsOnLink(dummyAddr, dummyLink)
	if !exists {
		t.Errorf("FAILED: Expected address to exist on link")
	}
}

func TestLookupFailedForRemoveAddressFromLink(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{simLookupFail: true}}
	err := nm.RemoveAddressFromLink("2001:db8:30::2/64", "eth3")
	if err == nil {
		t.Errorf("FAILED: Expected that link does not exist")
	}
	expectedErr := "Unable to find interface \"eth3\""
	if err.Error() != expectedErr {
		t.Errorf("FAILED: Expected failure message %q, got %q", expectedErr, err.Error())
	}
}

func TestParseFailedForRemoveAddressFromLink(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{simParseAddrFail: true}}
	err := nm.RemoveAddressFromLink("2001:db8::30::2/64", "eth3")
	if err == nil {
		t.Errorf("FAILED: Expected that address is invalid")
	}
	expectedErr := "Malformed address to delete \"2001:db8::30::2/64\""
	if err.Error() != expectedErr {
		t.Errorf("FAILED: Expected failure message %q, got %q", expectedErr, err.Error())
	}
}

func TestNotFoundForRemoveAddressFromLink(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
	err := nm.RemoveAddressFromLink("2001:db8:50::2/64", "eth2")
	if err == nil {
		t.Errorf("FAILED: Expected failure - no match for address")
	}
	expectedErr := "Skipping - address \"2001:db8:50::2/64\" does not exist on interface \"eth2\""
	if err.Error() != expectedErr {
		t.Errorf("FAILED: Expected failure message %q, got %q", expectedErr, err.Error())
	}
}

func TestRemoveAddressFromLink(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
	err := nm.RemoveAddressFromLink("2001:db8:20::2/64", "eth2")
	if err != nil {
		t.Errorf("FAILED: Expected success - matched address: %s", err.Error())
	}
}

func TestFailedDeleteForRemoveAddressFromLink(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{simDeleteFail: true}}
	err := nm.RemoveAddressFromLink("2001:db8:30::2/64", "eth3")
	if err == nil {
		t.Errorf("FAILED: Expected failure to remove address")
	}
	expectedErr := "Unable to delete ip \"2001:db8:30::2/64\" from interface \"eth3\""
	if err.Error() != expectedErr {
		t.Errorf("FAILED: Expected failure message %q, got %q", expectedErr, err.Error())
	}
}

func TestFindLinkIndexForCIDR(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
	idx, err := nm.FindLinkIndexForCIDR("2001:db8:30::3/64")
	if err != nil {
		t.Errorf("FAILED: Expected to find CIDR on link: %s", err.Error())
	}
	if idx != 30 {
		t.Errorf("FAILED: Expected to find CIDR on link with index 30, got link %d", idx)
	}
}

func TestFailedParseForFindLinkIndexForCIDR(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
	_, err := nm.FindLinkIndexForCIDR("2001:db8::30::2/64")
	if err == nil {
		t.Errorf("FAILED: Expected to fail parsing of CIDR")
	}
	expectedErr := "invalid CIDR address: 2001:db8::30::2/64"
	if err.Error() != expectedErr {
		t.Errorf("FAILED: Expected failure message %q, got %q", expectedErr, err.Error())
	}
}

func TestFailedNoLinkFindLinkIndexForCIDR(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{simLinkListFail: true}}
	_, err := nm.FindLinkIndexForCIDR("2001:db8:30::2/64")
	if err == nil {
		t.Errorf("FAILED: Expected no links")
	}
	expectedErr := "No links on system"
	if err.Error() != expectedErr {
		t.Errorf("FAILED: Expected failure message %q, got %q", expectedErr, err.Error())
	}
}

func TestFailedAddrNotFoundFindLinkIndexForCIDR(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
	_, err := nm.FindLinkIndexForCIDR("2001:db8:50::2/64")
	if err == nil {
		t.Errorf("FAILED: Expected not to find address on any links")
	}
	expectedErr := "Unable to find interface for CIDR \"2001:db8:50::2/64\""
	if err.Error() != expectedErr {
		t.Errorf("FAILED: Expected failure message %q, got %q", expectedErr, err.Error())
	}
}

func TestBuildRoute(t *testing.T) {
	r, err := lazyjack.BuildRoute("2001:db8:20::2/64", "2001:db8:20::1", 20)
	if err != nil {
		t.Errorf("FAILED: Expected to be able to build route: %s", err.Error())
	}

	expected := "2001:db8:20::/64"
	if r.Dst.String() != expected {
		t.Errorf("FAILED: Route destination wrong. Expeceted %q, got %q", expected, r.Dst.String())
	}
	expected = "2001:db8:20::1"
	if r.Gw.String() != expected {
		t.Errorf("FAILED: Route gateway wrong. Expected %q, got %q", expected, r.Gw.String())
	}
	expectedIdx := 20
	if r.LinkIndex != expectedIdx {
		t.Errorf("FAILED: Route gateway wrong. Expected %d, got %d", expectedIdx, r.LinkIndex)
	}
}

func TestFailedParseCIDRBuildRoute(t *testing.T) {
	_, err := lazyjack.BuildRoute("2001:db8::20::2/64", "2001:db8:20::1", 20)
	if err == nil {
		t.Errorf("FAILED: Expected to fail due to bad CIDR")
	}
	expected := "Unable to parse destination CIDR \"2001:db8::20::2/64\": invalid CIDR address: 2001:db8::20::2/64"
	if err.Error() != expected {
		t.Errorf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedParseIPBuildRoute(t *testing.T) {
	_, err := lazyjack.BuildRoute("2001:db8:20::2/64", "2001::db8:20::1", 20)
	if err == nil {
		t.Errorf("FAILED: Expected to fail due to bad GW IP")
	}
	expected := "Unable to parse gateway IP \"2001::db8:20::1\""
	if err.Error() != expected {
		t.Errorf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestAddRouteUsingSupportNetInterface(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
	err := nm.AddRouteUsingSupportNetInterface("2001:db8:30::2/64", "2001:db8:30::1", "2001:db8:20::2/64")
	if err != nil {
		t.Errorf("FAILED: Expected to be able to add route: %s", err.Error())
	}
}

func TestFailedAddRouteUsingSupportNetInterface(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{simRouteAddFail: true}}
	err := nm.AddRouteUsingSupportNetInterface("2001:db8:30::2/64", "2001:db8:30::1", "2001:db8:20::2/64")
	if err == nil {
		t.Errorf("FAILED: Expected to not be able to add route")
	}
	expected := "Mock failure adding route"
	if err.Error() != expected {
		t.Errorf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedBadCIDRAddRouteUsingSupportNetInterface(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
	err := nm.AddRouteUsingSupportNetInterface("2001:db8::30::2/64", "2001:db8:30::1", "2001:db8:20::2/64")
	if err == nil {
		t.Errorf("FAILED: Expected to not be able to add route")
	}
	expected := "Unable to parse destination CIDR \"2001:db8::30::2/64\": invalid CIDR address: 2001:db8::30::2/64"
	if err.Error() != expected {
		t.Errorf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedNotFoundAddRouteUsingSupportNetInterface(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
	err := nm.AddRouteUsingSupportNetInterface("2001:db8:30::2/64", "2001:db8:30::1", "2001:db8:50::2/64")
	if err == nil {
		t.Errorf("FAILED: Expected to not be able to find index from support net CIDR")
	}
	expected := "Unable to find interface for CIDR \"2001:db8:50::2/64\""
	if err.Error() != expected {
		t.Errorf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestDeleteRouteUsingSupportNetInterface(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
	err := nm.DeleteRouteUsingSupportNetInterface("2001:db8:30::2/64", "2001:db8:30::1", "2001:db8:20::2/64")
	if err != nil {
		t.Errorf("FAILED: Expected to be able to delete route: %s", err.Error())
	}
}

func TestFailedDeleteRouteUsingSupportNetInterface(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{simRouteDelFail: true}}
	err := nm.DeleteRouteUsingSupportNetInterface("2001:db8:30::2/64", "2001:db8:30::1", "2001:db8:20::2/64")
	if err == nil {
		t.Errorf("FAILED: Expected to not be able to delete route")
	}
	expected := "Mock failure deleting route"
	if err.Error() != expected {
		t.Errorf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedBadCIDRDeleteRouteUsingSupportNetInterface(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
	err := nm.DeleteRouteUsingSupportNetInterface("2001:db8::30::2/64", "2001:db8:30::1", "2001:db8:20::2/64")
	if err == nil {
		t.Errorf("FAILED: Expected to not be able to delete route")
	}
	expected := "Unable to parse destination CIDR \"2001:db8::30::2/64\": invalid CIDR address: 2001:db8::30::2/64"
	if err.Error() != expected {
		t.Errorf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedNotFoundDeleteRouteUsingSupportNetInterface(t *testing.T) {
	nm := &lazyjack.NetManager{Mgr: &mockImpl{}}
	err := nm.DeleteRouteUsingSupportNetInterface("2001:db8:30::2/64", "2001:db8:30::1", "2001:db8:50::2/64")
	if err == nil {
		t.Errorf("FAILED: Expected to not be able to find index from support net CIDR")
	}
	expected := "Unable to find interface for CIDR \"2001:db8:50::2/64\""
	if err.Error() != expected {
		t.Errorf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestBuildGWIP(t *testing.T) {
	actual := lazyjack.BuildGWIP("2001:db8::", 5)
	expected := "2001:db8::5"
	if actual != expected {
		t.Errorf("FAILED: Gateway IP create. Expected %q, got %q", expected, actual)
	}
}
