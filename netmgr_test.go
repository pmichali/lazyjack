package lazyjack_test

import (
	"fmt"
	"net"
	"strconv"
	"testing"

	"github.com/pmichali/lazyjack"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netlink/nl"
)

type mockNetLink struct {
	simLookupFail    bool
	simParseAddrFail bool
	simDeleteFail    bool
	simReplaceFail   bool
	simReplaceFail2  bool
	simAddrListFail  bool
	simLinkListFail  bool
	simRouteAddFail  bool
	simRouteExists   bool
	simRouteDelFail  bool
	simNoRoute       bool
	simSetDownFail   bool
	simLinkDelFail   bool
	simSetMTUFail    bool
	callCount        int
}

func (m *mockNetLink) ResetCallCount() {
	m.callCount = 0
}

func (m *mockNetLink) CallCount() int {
	return m.callCount
}

func (m *mockNetLink) Called() {
	m.callCount++
}

func (m *mockNetLink) AddrDel(link netlink.Link, addr *netlink.Addr) error {
	if m.simDeleteFail {
		return fmt.Errorf("mock failure to delete address")
	}
	return nil
}

func (m *mockNetLink) AddrList(link netlink.Link, family int) ([]netlink.Addr, error) {
	if m.simAddrListFail {
		return []netlink.Addr{}, fmt.Errorf("mock failure to list addresses")
	}
	addrList := []netlink.Addr{}
	// Will use the link index to create dummy addresses per link
	var addr *netlink.Addr
	if family == nl.FAMILY_V4 || family == nl.FAMILY_ALL {
		addr, _ = netlink.ParseAddr(fmt.Sprintf("172.%d.0.2/16", link.Attrs().Index))
		addrList = append(addrList, *addr)
		addr, _ = netlink.ParseAddr(fmt.Sprintf("172.%d.0.3/16", link.Attrs().Index))
		addrList = append(addrList, *addr)
		addr, _ = netlink.ParseAddr(fmt.Sprintf("172.%d.0.4/16", link.Attrs().Index))
		addrList = append(addrList, *addr)
		// To simulate mgmt IP address, which has node ID as last part of address
		addr, _ = netlink.ParseAddr(fmt.Sprintf("10.192.0.%d/16", link.Attrs().Index))
		addrList = append(addrList, *addr)
	}
	if family == nl.FAMILY_V6 || family == nl.FAMILY_ALL {
		addr, _ = netlink.ParseAddr(fmt.Sprintf("2001:db8:%x::2/64", link.Attrs().Index))
		addrList = append(addrList, *addr)
		addr, _ = netlink.ParseAddr(fmt.Sprintf("2001:db8:%x::3/64", link.Attrs().Index))
		addrList = append(addrList, *addr)
		// To simulate mgmt IP address, which has node ID as last part of address
		addr, _ = netlink.ParseAddr(fmt.Sprintf("2001:db8:20::%x/64", link.Attrs().Index))
		addrList = append(addrList, *addr)
	}
	return addrList, nil
}

func (m *mockNetLink) AddrReplace(link netlink.Link, addr *netlink.Addr) error {
	if m.simReplaceFail {
		return fmt.Errorf("mock failure to replace address")
	}
	if m.simReplaceFail2 && m.CallCount() == 1 {
		return fmt.Errorf("mock failure to replace second address")
	}
	m.Called()
	return nil
}

func (m *mockNetLink) LinkByName(name string) (netlink.Link, error) {
	if m.simLookupFail {
		return nil, fmt.Errorf("mock failure to find link")
	}
	// Calc index based on interface name, using last digit * 16.
	// For example "eth2" -> 2*16 = 0x20.
	link := &netlink.Device{}
	idx, _ := strconv.Atoi(name[len(name)-1:])
	link.Index = idx * 16
	return link, nil
}

func (m *mockNetLink) LinkList() ([]netlink.Link, error) {
	if m.simLinkListFail {
		return []netlink.Link{}, fmt.Errorf("mock failure to list addresses")
	}
	// Making a dummy list of two entries with indexes 0x20 and 0x30. The dummy addresses
	// we create for some tests, will use the index as part of the IP.
	linkA := &netlink.Device{}
	linkA.Index = 0x20
	linkB := &netlink.Device{}
	linkB.Index = 0x30
	return []netlink.Link{linkA, linkB}, nil
}

func (m *mockNetLink) ParseAddr(s string) (*netlink.Addr, error) {
	if m.simParseAddrFail {
		return nil, fmt.Errorf("mock failure to parse address")
	}
	return netlink.ParseAddr(s)
}

func (m *mockNetLink) ParseIPNet(s string) (*net.IPNet, error) {
	return netlink.ParseIPNet(s)
}

func (m *mockNetLink) RouteAdd(route *netlink.Route) error {
	if m.simRouteAddFail {
		return fmt.Errorf("mock failure adding route")
	}
	if m.simRouteExists {
		return fmt.Errorf("file exists")
	}
	m.Called()
	return nil
}

func (m *mockNetLink) RouteDel(route *netlink.Route) error {
	if m.simRouteDelFail {
		return fmt.Errorf("mock failure deleting route")
	}
	if m.simNoRoute {
		return fmt.Errorf("no such process")
	}
	m.Called()
	return nil
}

func (m *mockNetLink) LinkSetDown(link netlink.Link) error {
	if m.simSetDownFail {
		return fmt.Errorf("mock failure set link down")
	}
	return nil
}

func (m *mockNetLink) LinkDel(link netlink.Link) error {
	if m.simLinkDelFail {
		return fmt.Errorf("mock failure link delete")
	}
	return nil
}

func (m *mockNetLink) LinkSetMTU(link netlink.Link, mtu int) error {
	if m.simSetMTUFail {
		return fmt.Errorf("mock failure set link MTU")
	}
	return nil
}

func TestAddAddressToLink(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{}}
	err := nm.AddAddressToLink("2001:db8::10/64", "eth1")
	if err != nil {
		t.Fatalf("FAILED: Expected address add to pass: %s", err.Error())
	}
}

func TestBuildNodeCIDR(t *testing.T) {
	info := lazyjack.NetInfo{Prefix: "2001:db8:20::", Mode: lazyjack.IPv6NetMode, Size: 64}
	actual := lazyjack.BuildNodeCIDR(info, 2)
	expected := "2001:db8:20::2/64"
	if actual != expected {
		t.Fatalf("FAILED: Node CIDR create. Expected %q, got %q", expected, actual)
	}
	info = lazyjack.NetInfo{Prefix: "10.20.0.", Mode: lazyjack.IPv4NetMode, Size: 16}
	actual = lazyjack.BuildNodeCIDR(info, 2)
	expected = "10.20.0.2/16"
	if actual != expected {
		t.Fatalf("FAILED: Node CIDR create. Expected %q, got %q", expected, actual)
	}
}

func TestFailedLookupForAddAddressToLink(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{simLookupFail: true}}
	err := nm.AddAddressToLink("2001:db8::10", "eth1")
	if err == nil {
		t.Fatalf("FAILED: Expected address add to fail")
	}
	expectedErr := "unable to find interface \"eth1\""
	if err.Error() != expectedErr {
		t.Fatalf("FAILED: Expected failure message %q, got %q", expectedErr, err.Error())
	}
}

func TestFailedParseForAddAddressToLink(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{simParseAddrFail: true}}
	err := nm.AddAddressToLink("2001::db8::10/64", "eth1")
	if err == nil {
		t.Fatalf("FAILED: Expected address add to fail")
	}
	expectedErr := "malformed address \"2001::db8::10/64\""
	if err.Error() != expectedErr {
		t.Fatalf("FAILED: Expected failure message %q, got %q", expectedErr, err.Error())
	}
}

func TestFailedReplaceForAddAddressToLink(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{simReplaceFail: true}}
	err := nm.AddAddressToLink("2001:db8::10/64", "eth1")
	if err == nil {
		t.Fatalf("FAILED: Expected address add to fail")
	}
	expectedErr := "unable to add ip \"2001:db8::10/64\" to interface \"eth1\""
	if err.Error() != expectedErr {
		t.Fatalf("FAILED: Expected failure message %q, got %q", expectedErr, err.Error())
	}
}

func TestFailureAddressExistsOnLink(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{simAddrListFail: true}}
	dummyAddr, _ := netlink.ParseAddr("2001:db8:10::2/64")
	dummyLink := &netlink.Device{}
	dummyLink.Index = 10
	exists := nm.AddressExistsOnLink(dummyAddr, dummyLink)
	if exists {
		t.Fatalf("FAILED: Expected address to not exist on link")
	}
}

func TestNotFoundAddressExistsOnLink(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{}}
	dummyAddr, _ := netlink.ParseAddr("2001:db8:50::2/64")
	dummyLink := &netlink.Device{}
	dummyLink.Index = 0x10
	exists := nm.AddressExistsOnLink(dummyAddr, dummyLink)
	if exists {
		t.Fatalf("FAILED: Expected address to not exist on link")
	}
}

func TestAddressExistsOnLink(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{}}
	dummyAddr, _ := netlink.ParseAddr("2001:db8:30::2/64")
	dummyLink := &netlink.Device{}
	dummyLink.Index = 0x30
	exists := nm.AddressExistsOnLink(dummyAddr, dummyLink)
	if !exists {
		t.Fatalf("FAILED: Expected address to exist on link")
	}
}

func TestLookupFailedForRemoveAddressFromLink(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{simLookupFail: true}}
	err := nm.RemoveAddressFromLink("2001:db8:30::2/64", "eth3")
	if err == nil {
		t.Fatalf("FAILED: Expected that link does not exist")
	}
	expectedErr := "unable to find interface \"eth3\""
	if err.Error() != expectedErr {
		t.Fatalf("FAILED: Expected failure message %q, got %q", expectedErr, err.Error())
	}
}

func TestParseFailedForRemoveAddressFromLink(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{simParseAddrFail: true}}
	err := nm.RemoveAddressFromLink("2001:db8::30::2/64", "eth3")
	if err == nil {
		t.Fatalf("FAILED: Expected that address is invalid")
	}
	expectedErr := "malformed address to delete \"2001:db8::30::2/64\""
	if err.Error() != expectedErr {
		t.Fatalf("FAILED: Expected failure message %q, got %q", expectedErr, err.Error())
	}
}

func TestNotFoundForRemoveAddressFromLink(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{}}
	err := nm.RemoveAddressFromLink("2001:db8:50::2/64", "eth2")
	if err == nil {
		t.Fatalf("FAILED: Expected failure - no match for address")
	}
	expectedErr := "skipping - address \"2001:db8:50::2/64\" does not exist on interface \"eth2\""
	if err.Error() != expectedErr {
		t.Fatalf("FAILED: Expected failure message %q, got %q", expectedErr, err.Error())
	}
}

func TestRemoveAddressFromLink(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{}}
	err := nm.RemoveAddressFromLink("2001:db8:20::2/64", "eth2")
	if err != nil {
		t.Fatalf("FAILED: Expected success - matched address: %s", err.Error())
	}
}

func TestFailedDeleteForRemoveAddressFromLink(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{simDeleteFail: true}}
	err := nm.RemoveAddressFromLink("2001:db8:30::2/64", "eth3")
	if err == nil {
		t.Fatalf("FAILED: Expected failure to remove address")
	}
	expectedErr := "unable to delete ip \"2001:db8:30::2/64\" from interface \"eth3\""
	if err.Error() != expectedErr {
		t.Fatalf("FAILED: Expected failure message %q, got %q", expectedErr, err.Error())
	}
}

func TestFindLinkIndexForCIDR(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{}}
	idx, err := nm.FindLinkIndexForCIDR("172.32.0.0/16")
	if err != nil {
		t.Fatalf("FAILED: Expected to find CIDR on link: %s", err.Error())
	}
	if idx != 32 {
		t.Fatalf("FAILED: Expected to find CIDR on link with index 32, got link %d", idx)
	}
}

func TestFailedParseForFindLinkIndexForCIDR(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{}}
	_, err := nm.FindLinkIndexForCIDR("172.30.0.0.0/16")
	if err == nil {
		t.Fatalf("FAILED: Expected to fail parsing of CIDR")
	}
	expectedErr := "invalid CIDR address: 172.30.0.0.0/16"
	if err.Error() != expectedErr {
		t.Fatalf("FAILED: Expected failure message %q, got %q", expectedErr, err.Error())
	}
}

func TestFailedNoLinkFindLinkIndexForCIDR(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{simLinkListFail: true}}
	_, err := nm.FindLinkIndexForCIDR("172.30.0.0/16")
	if err == nil {
		t.Fatalf("FAILED: Expected no links")
	}
	expectedErr := "no links on system"
	if err.Error() != expectedErr {
		t.Fatalf("FAILED: Expected failure message %q, got %q", expectedErr, err.Error())
	}
}

func TestFailedAddrNotFoundFindLinkIndexForCIDR(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{}}
	_, err := nm.FindLinkIndexForCIDR("172.50.0.0/16")
	if err == nil {
		t.Fatalf("FAILED: Expected not to find address on any links")
	}
	expectedErr := "unable to find interface for CIDR \"172.50.0.0/16\""
	if err.Error() != expectedErr {
		t.Fatalf("FAILED: Expected failure message %q, got %q", expectedErr, err.Error())
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
		t.Fatalf("FAILED: Expected to fail due to bad CIDR")
	}
	expected := "unable to parse destination CIDR \"2001:db8::20::2/64\": invalid CIDR address: 2001:db8::20::2/64"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedParseIPBuildRoute(t *testing.T) {
	_, err := lazyjack.BuildRoute("2001:db8:20::2/64", "2001::db8:20::1", 20)
	if err == nil {
		t.Fatalf("FAILED: Expected to fail due to bad GW IP")
	}
	expected := "unable to parse gateway IP \"2001::db8:20::1\""
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestAddRouteUsingSupportNetInterface(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{}}
	err := nm.AddRouteUsingSupportNetInterface("2001:db8:30::2/64", "2001:db8:30::1", "172.32.0.0/16")
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to add route: %s", err.Error())
	}
}

func TestFailedAddRouteUsingSupportNetInterface(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{simRouteAddFail: true}}
	err := nm.AddRouteUsingSupportNetInterface("2001:db8:30::2/64", "2001:db8:30::1", "172.32.0.0/16")
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to add route")
	}
	expected := "mock failure adding route"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedBadCIDRAddRouteUsingSupportNetInterface(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{}}
	err := nm.AddRouteUsingSupportNetInterface("2001:db8::30::2/64", "2001:db8:30::1", "172.32.0.0/16")
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to add route")
	}
	expected := "unable to parse destination CIDR \"2001:db8::30::2/64\": invalid CIDR address: 2001:db8::30::2/64"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedNotFoundAddRouteUsingSupportNetInterface(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{}}
	err := nm.AddRouteUsingSupportNetInterface("2001:db8:30::2/64", "2001:db8:30::1", "172.50.0.0/16")
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to find index from support net CIDR")
	}
	expected := "unable to find interface for CIDR \"172.50.0.0/16\""
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestDeleteRouteUsingSupportNetInterface(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{}}
	err := nm.DeleteRouteUsingSupportNetInterface("2001:db8:30::2/64", "2001:db8:30::1", "172.32.0.0/16")
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to delete route: %s", err.Error())
	}
}

func TestFailedDeleteRouteUsingSupportNetInterface(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{simRouteDelFail: true}}
	err := nm.DeleteRouteUsingSupportNetInterface("2001:db8:30::2/64", "2001:db8:30::1", "172.32.0.0/16")
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to delete route")
	}
	expected := "mock failure deleting route"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedBadCIDRDeleteRouteUsingSupportNetInterface(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{}}
	err := nm.DeleteRouteUsingSupportNetInterface("2001:db8::30::2/64", "2001:db8:30::1", "172.30.40.0.0/16")
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to delete route")
	}
	expected := "invalid CIDR address: 172.30.40.0.0/16"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedNotFoundDeleteRouteUsingSupportNetInterface(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{}}
	err := nm.DeleteRouteUsingSupportNetInterface("2001:db8:30::2/64", "2001:db8:30::1", "172.50.0.0/16")
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to find index from support net CIDR")
	}
	expected := "unable to find interface for CIDR \"172.50.0.0/16\""
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedBadGWDeleteRouteUsingSupportNetInterface(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{}}
	err := nm.DeleteRouteUsingSupportNetInterface("2001:db8:30::2/64", "2001::db8:30::1", "172.32.0.0/16")
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to delete route because of invalid GW")
	}
	expected := "unable to parse gateway IP \"2001::db8:30::1\""
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestBuildGWIP(t *testing.T) {
	actual := lazyjack.BuildGWIP("2001:db8::", 5)
	expected := "2001:db8::5"
	if actual != expected {
		t.Fatalf("FAILED: Gateway IP create. Expected %q, got %q", expected, actual)
	}
}

func TestAddRouteUsingInterfaceName(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{}}
	err := nm.AddRouteUsingInterfaceName("2001:db8:30::2/64", "2001:db8:30::1", "eth3")
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to add route: %s", err.Error())
	}
}

func TestFailedAddRouteUsingInterfaceName(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{simRouteAddFail: true}}
	err := nm.AddRouteUsingInterfaceName("2001:db8:30::2/64", "2001:db8:30::1", "eth3")
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to add route")
	}
	expected := "mock failure adding route"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedBadCIDRAddRouteUsingInterfaceName(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{}}
	err := nm.AddRouteUsingInterfaceName("2001:db8::30::2/64", "2001:db8:30::1", "eth3")
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to add route")
	}
	expected := "unable to parse destination CIDR \"2001:db8::30::2/64\": invalid CIDR address: 2001:db8::30::2/64"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedNotFoundAddRouteUsingInterfaceName(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{simLookupFail: true}}
	err := nm.AddRouteUsingInterfaceName("2001:db8:30::2/64", "2001:db8:30::1", "eth3")
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to find index for link")
	}
	expected := "unable to find interface \"eth3\""
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedExistsAddRouteUsingInterfaceName(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{simRouteExists: true}}
	err := nm.AddRouteUsingInterfaceName("2001:db8:30::2/64", "2001:db8:30::1", "eth3")
	if err == nil {
		t.Fatalf("FAILED: Expected failure due to existing route")
	}
	expected := "file exists"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestDeleteRouteUsingInterfaceName(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{}}
	err := nm.DeleteRouteUsingInterfaceName("2001:db8:30::2/64", "2001:db8:30::1", "eth3")
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to delete route: %s", err.Error())
	}
}

func TestFailedDeleteRouteUsingInterfaceName(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{simRouteDelFail: true}}
	err := nm.DeleteRouteUsingInterfaceName("2001:db8:30::2/64", "2001:db8:30::1", "eth3")
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to delete route")
	}
	expected := "mock failure deleting route"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedBadCIDRDeleteRouteUsingInterfaceName(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{}}
	err := nm.DeleteRouteUsingInterfaceName("2001:db8::30::2/64", "2001:db8:30::1", "eth3")
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to delete route")
	}
	expected := "unable to parse destination CIDR \"2001:db8::30::2/64\": invalid CIDR address: 2001:db8::30::2/64"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedNotFoundDeleteRouteUsingInterfaceName(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{simLookupFail: true}}
	err := nm.DeleteRouteUsingInterfaceName("2001:db8:30::2/64", "2001:db8:30::1", "eth3")
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to find index for link")
	}
	expected := "skipping - Unable to find interface \"eth3\" to delete route"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestBringLinkDown(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{}}
	err := nm.BringLinkDown("br0")
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to bring link down")
	}
}

func TestFailedBringLinkDown(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{simSetDownFail: true}}
	err := nm.BringLinkDown("br0")
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to bring link down")
	}
	expected := "unable to shut down interface \"br0\""
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedNotFoundBringLinkDown(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{simLookupFail: true}}
	err := nm.BringLinkDown("br0")
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to find link to bring down")
	}
	expected := "unable to find interface \"br0\""
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestDeleteLink(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{}}
	err := nm.DeleteLink("br0")
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to delete link")
	}
}

func TestFailedDeleteLink(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{simLinkDelFail: true}}
	err := nm.DeleteLink("br0")
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to delete link")
	}
	expected := "unable to delete interface \"br0\""
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedNotFoundDeleteLink(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{simLookupFail: true}}
	err := nm.DeleteLink("br0")
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to find link to delete")
	}
	expected := "unable to find interface \"br0\""
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestSetLinkMTU(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{}}
	err := nm.SetLinkMTU("eth0", 1500)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to set MTU")
	}
}

func TestFailedSetLinkMTU(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{simSetMTUFail: true}}
	err := nm.SetLinkMTU("eth0", 1500)
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to set MTU")
	}
	expected := "unable to set MTU on interface \"eth0\""
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestFailedNotFoundSetLinkMTU(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{simLookupFail: true}}
	err := nm.SetLinkMTU("eth0", 9000)
	if err == nil {
		t.Fatalf("FAILED: Expected to not be able to find link to set MTU")
	}
	expected := "unable to find interface \"eth0\""
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg %q, got %q", expected, err.Error())
	}
}

func TestRemoveBridge(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{}}
	err := nm.RemoveBridge("br0")
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to remove bridge: %s", err.Error())
	}
}

func TestFailedLinkDownRemoveBridge(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{simSetDownFail: true}}
	err := nm.RemoveBridge("br0")
	if err == nil {
		t.Fatalf("FAILED: Expected to fail bringing link down")
	}
	expected := "unable to shut down interface \"br0\""
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg to start with %q, got %q", expected, err.Error())
	}
}

func TestFailedLinkDeleteRemoveBridge(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{simLinkDelFail: true}}
	err := nm.RemoveBridge("br0")
	if err == nil {
		t.Fatalf("FAILED: Expected to fail deleting link")
	}
	expected := "unable to delete interface \"br0\""
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg to start with %q, got %q", expected, err.Error())
	}
}

func TestFailedAllRemoveBridge(t *testing.T) {
	nm := lazyjack.NetMgr{Server: &mockNetLink{simSetDownFail: true, simLinkDelFail: true}}
	err := nm.RemoveBridge("br0")
	if err == nil {
		t.Fatalf("FAILED: Expected to fail bringing link down and deleting link")
	}
	expected := "unable to bring link down (unable to shut down interface \"br0\"), nor remove link (unable to delete interface \"br0\")"
	if err.Error() != expected {
		t.Fatalf("FAILED: Expected msg to start with %q, got %q", expected, err.Error())
	}
}
