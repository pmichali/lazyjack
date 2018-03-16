package lazyjack_test

import (
	"strings"
	"testing"

	"github.com/pmichali/lazyjack"
)

func TestBuildDockerArgsForDNS64(t *testing.T) {
	c := &lazyjack.Config{
		DNS64: lazyjack.DNS64Config{ServerIP: "2001:db8::100"},
		General: lazyjack.GeneralSettings{
			WorkArea: "/tmp/lazyjack",
		},
	}

	list := lazyjack.BuildRunArgsForDNS64(c)
	actual := strings.Join(list, " ")
	expected := "run -d --name bind9 --hostname bind9 --label lazyjack --privileged=true --ip6 2001:db8::100 --dns 2001:db8::100 --sysctl net.ipv6.conf.all.disable_ipv6=0 --sysctl net.ipv6.conf.all.forwarding=1 -v /tmp/lazyjack/bind9/conf/named.conf:/etc/bind/named.conf --net support_net resystit/bind9:latest"
	if actual != expected {
		t.Fatalf("FAILED: Building docker run args for DNS64.\nExpected: %q\n  Actual: %q", expected, actual)
	}
}

func TestBuildCreateSupportNetArgs(t *testing.T) {
	list := lazyjack.BuildCreateNetArgsForSupportNet("fd00:10::/64", "fd00:10::", "172.18.0.0/16")
	actual := strings.Join(list, " ")
	expected := "network create --ipv6 --subnet=\"fd00:10::/64\" --subnet=172.18.0.0/16 --gateway=\"fd00:10::1\" support_net"
	if actual != expected {
		t.Fatalf("FAILED: Building support net create args. Expected %q, got %q", expected, actual)
	}
}

func TestBuildDeleteSupportNetArgs(t *testing.T) {
	list := lazyjack.BuildDeleteNetArgsForSupportNet()
	actual := strings.Join(list, " ")
	expected := "network rm support_net"
	if actual != expected {
		t.Fatalf("FAILED: Building support net delete args. Expected %q, got %q", expected, actual)
	}
}

func TestBuildGetInterfaceArgs(t *testing.T) {
	list := lazyjack.BuildGetInterfaceArgsForDNS64()
	actual := strings.Join(list, " ")
	expected := "exec bind9 ip addr list eth0"
	if actual != expected {
		t.Fatalf("FAILED: Building eth0 I/F config args. Expected %q, got %q", expected, actual)
	}
}

func TestBuildAddrDeleteArgs(t *testing.T) {
	list := lazyjack.BuildV4AddrDelArgsForDNS64("172.18.0.2/16")
	actual := strings.Join(list, " ")
	expected := "exec bind9 ip addr del 172.18.0.2/16 dev eth0"
	if actual != expected {
		t.Fatalf("FAILED: Building I/F delete args. Expected %q, got %q", expected, actual)
	}
}

func TestBuildAddRouteForDNS64Args(t *testing.T) {
	c := &lazyjack.Config{
		DNS64: lazyjack.DNS64Config{
			CIDR:       "fd00:10:64:ff9b::/96",
			CIDRPrefix: "fd00:10:64:ff9b::",
		},
		NAT64: lazyjack.NAT64Config{
			ServerIP: "fd00:10::200",
		},
	}

	list := lazyjack.BuildAddRouteArgsForDNS64(c)
	actual := strings.Join(list, " ")
	expected := "exec bind9 ip -6 route add fd00:10:64:ff9b::/96 via fd00:10::200"
	if actual != expected {
		t.Fatalf("FAILED: Building add route args. Expected %q, got %q", expected, actual)
	}
}

func TestBuildDockerArgsForNAT64(t *testing.T) {
	c := &lazyjack.Config{
		DNS64: lazyjack.DNS64Config{
			CIDR:           "fd00:10:64:ff9b::/96",
			CIDRPrefix:     "fd00:10:64:ff9b::",
			ServerIP:       "fd00:10::100",
			RemoteV4Server: "8.8.8.8",
		},
		NAT64: lazyjack.NAT64Config{
			ServerIP:    "fd00:10::200",
			V4MappingIP: "172.18.0.200",
		},
	}

	list := lazyjack.BuildRunArgsForNAT64(c)
	actual := strings.Join(list, " ")
	expected := "run -d --name tayga --hostname tayga --label lazyjack --privileged=true --ip 172.18.0.200 --ip6 fd00:10::200 --dns 8.8.8.8 --dns fd00:10::100 --sysctl net.ipv6.conf.all.disable_ipv6=0 --sysctl net.ipv6.conf.all.forwarding=1 -e TAYGA_CONF_PREFIX=fd00:10:64:ff9b::/96 -e TAYGA_CONF_IPV4_ADDR=172.18.0.200 --net support_net danehans/tayga:latest"
	if actual != expected {
		t.Fatalf("FAILED: Building docker run args for NAT64.\nExpected: %q\n  Actual: %q", expected, actual)
	}
}
