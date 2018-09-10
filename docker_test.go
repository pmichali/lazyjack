package lazyjack_test

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/pmichali/lazyjack"
)

func TestBuildResourceStateArgs(t *testing.T) {
	list := lazyjack.BuildResourceStateArgs("my-network")
	actual := strings.Join(list, " ")
	expected := "inspect my-network"
	if actual != expected {
		t.Fatalf("FAILED: Building resource state args. Expected %q, got %q", expected, actual)
	}
}

func TestResourceState(t *testing.T) {
	var testCases = []struct {
		name     string
		resource string
		cmd      string
		expected string
	}{
		{
			name:     "resource exists",
			resource: "host", // Should be there on system with docker installed
			cmd:      lazyjack.DefaultDockerCommand,
			expected: lazyjack.ResourceExists,
		},
		{
			name:     "resource running",
			resource: "\"Running\": true", // Bogus name so that output has expected string when doing echo, instead of docker command
			cmd:      "echo",
			expected: lazyjack.ResourceRunning,
		},
		{
			name:     "resource doesn't exists",
			resource: "no-such-resource",
			cmd:      lazyjack.DefaultDockerCommand,
			expected: lazyjack.ResourceNotPresent,
		},
	}
	for _, tc := range testCases {
		d := lazyjack.Docker{Command: tc.cmd}
		actual := d.ResourceState(tc.resource)
		if actual != tc.expected {
			t.Fatalf("FAILED: [%s] resource state mismatch. Expected %q, got %q", tc.name, tc.expected, actual)
		}
	}
}

func TestBuildDockerArgsForDNS64(t *testing.T) {
	c := &lazyjack.Config{
		DNS64: lazyjack.DNS64Config{ServerIP: "2001:db8::100"},
		General: lazyjack.GeneralSettings{
			WorkArea: "/tmp/lazyjack",
		},
	}

	list := lazyjack.BuildRunArgsForDNS64(c)
	actual := strings.Join(list, " ")
	expected := "run -d --name bind9 --hostname bind9 --label lazyjack --privileged=true --ip6 2001:db8::100 --dns 2001:db8::100 --sysctl net.ipv6.conf.all.disable_ipv6=0 --sysctl net.ipv6.conf.all.forwarding=1 -v volume-bind9:/etc/bind/ --net support_net diverdane/bind9:latest"
	if actual != expected {
		t.Fatalf("FAILED: Building docker run args for DNS64.\nExpected: %q\n  Actual: %q", expected, actual)
	}
}

func TestBuildCreateSupportNetArgs(t *testing.T) {
	list := lazyjack.BuildCreateNetArgsFor("test_net", "fd00:10::/64", "172.18.0.0/16", "fd00:10::")
	actual := strings.Join(list, " ")
	expected := "network create --ipv6 --subnet=\"fd00:10::/64\" --subnet=172.18.0.0/16 --gateway=\"fd00:10::1\" test_net"
	if actual != expected {
		t.Fatalf("FAILED: Building support net create args. Expected %q, got %q", expected, actual)
	}
}

func TestCreateNetwork(t *testing.T) {
	d := lazyjack.Docker{Command: "echo"}
	err := d.CreateNetwork("my-network", "2001:db8::/64", "10.20.0.0/16", "2001:db8::")
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to create network")
	}
}

func TestBuildDeleteNetArgs(t *testing.T) {
	list := lazyjack.BuildDeleteNetArgsFor(lazyjack.SupportNetName)
	actual := strings.Join(list, " ")
	expected := "network rm support_net"
	if actual != expected {
		t.Fatalf("FAILED: Building support net delete args. Expected %q, got %q", expected, actual)
	}
}

func TestDeleteNetwork(t *testing.T) {
	d := lazyjack.Docker{Command: "echo"}
	err := d.DeleteNetwork("my-network")
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to delete network")
	}
}

func TestBuildGetInterfaceArgs(t *testing.T) {
	list := lazyjack.BuildGetInterfaceArgs("bind9", "eth0")
	actual := strings.Join(list, " ")
	expected := "exec bind9 ip addr list eth0"
	if actual != expected {
		t.Fatalf("FAILED: Building eth0 I/F config args. Expected %q, got %q", expected, actual)
	}
}

func TestGetInterfaceConfig(t *testing.T) {
	d := lazyjack.Docker{Command: "echo"}
	actual, err := d.GetInterfaceConfig("my-container", "eth0")
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to get interface config")
	}
	expected := "exec my-container ip addr list eth0\n"
	if actual != expected {
		t.Fatalf("FAILED: Get I/F config. Expected %q, got %q", expected, actual)
	}
}

func TestBuildAddrDeleteArgs(t *testing.T) {
	list := lazyjack.BuildV4AddrDelArgs("bind9", "172.18.0.2/16")
	actual := strings.Join(list, " ")
	expected := "exec bind9 ip addr del 172.18.0.2/16 dev eth0"
	if actual != expected {
		t.Fatalf("FAILED: Building I/F delete args. Expected %q, got %q", expected, actual)
	}
}

func TestDeleteV4Address(t *testing.T) {
	d := lazyjack.Docker{Command: "echo"}
	err := d.DeleteV4Address("my-container", "192.168.0.5/24")
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to delete V4 IP")
	}
}

func TestBuildAddRouteForDNS64Args(t *testing.T) {
	list := lazyjack.BuildAddRouteArgs("bind9", "fd00:10:64:ff9b::/96", "fd00:10::200")
	actual := strings.Join(list, " ")
	expected := "exec bind9 ip -6 route add fd00:10:64:ff9b::/96 via fd00:10::200"
	if actual != expected {
		t.Fatalf("FAILED: Building add route args. Expected %q, got %q", expected, actual)
	}
}

func TestAddV6Route(t *testing.T) {
	d := lazyjack.Docker{Command: "echo"}
	err := d.AddV6Route("my-container", "2001:db8::/64", "2001:db8::200")
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to add route")
	}
}

func TestBuildDeleteContainerArgs(t *testing.T) {
	list := lazyjack.BuildDeleteContainerArgs("bind9")
	actual := strings.Join(list, " ")
	expected := "rm -f bind9"
	if actual != expected {
		t.Fatalf("FAILED: Building delete container args. Expected %q, got %q", expected, actual)
	}
}

func TestDeleteContainer(t *testing.T) {
	d := lazyjack.Docker{Command: "echo"}
	err := d.DeleteContainer("my-container")
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to delete container")
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
			ServerIP:      "fd00:10::200",
			V4MappingIP:   "172.18.0.200",
			V4MappingCIDR: "172.18.0.128/25",
		},
	}

	list := lazyjack.BuildRunArgsForNAT64(c)
	actual := strings.Join(list, " ")
	expected := "run -d --name tayga --hostname tayga --label lazyjack --privileged=true --ip 172.18.0.200 --ip6 fd00:10::200 --dns 8.8.8.8 --dns fd00:10::100 --sysctl net.ipv6.conf.all.disable_ipv6=0 --sysctl net.ipv6.conf.all.forwarding=1 -e TAYGA_CONF_PREFIX=fd00:10:64:ff9b::/96 -e TAYGA_CONF_IPV4_ADDR=172.18.0.200 -e TAYGA_CONF_DYNAMIC_POOL=172.18.0.128/25 --net support_net danehans/tayga:latest"
	if actual != expected {
		t.Fatalf("FAILED: Building docker run args for NAT64.\nExpected: %q\n  Actual: %q", expected, actual)
	}
}

func TestRunContainer(t *testing.T) {
	d := lazyjack.Docker{Command: "echo"}
	err := d.RunContainer("my-container", []string{"arg1", "arg2"})
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to run container")
	}
}

func TestBuildCreateVolumeArgs(t *testing.T) {
	list := lazyjack.BuildCreateVolumeArgs("volume-dummy")
	actual := strings.Join(list, " ")
	expected := "volume create volume-dummy"
	if actual != expected {
		t.Fatalf("FAILED: Building create volume args. Expected %q, got %q", expected, actual)
	}
}

func TestCreateVolume(t *testing.T) {
	d := lazyjack.Docker{Command: "echo"}
	err := d.CreateVolume("my-volume")
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to create volume")
	}
}

func TestBuildDeleteVolumeArgs(t *testing.T) {
	list := lazyjack.BuildDeleteVolumeArgs("volume-dummy")
	actual := strings.Join(list, " ")
	expected := "volume rm -f volume-dummy"
	if actual != expected {
		t.Fatalf("FAILED: Building delete volume args. Expected %q, got %q", expected, actual)
	}
}

func TestDeleteVolume(t *testing.T) {
	d := lazyjack.Docker{Command: "echo"}
	err := d.DeleteVolume("my-volume")
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to delete volume")
	}
}

func TestBuildInspectVolumeArgs(t *testing.T) {
	list := lazyjack.BuildInspectVolumeArgs("volume-dummy")
	actual := strings.Join(list, " ")
	expected := "volume inspect -f \"{{json .Mountpoint}}\" volume-dummy"
	if actual != expected {
		t.Fatalf("FAILED: Building inspect volume args. Expected %q, got %q", expected, actual)
	}
}

func TestGetVolumeMountPoint(t *testing.T) {
	// Need to do a real docker command here, because the result is post processed some
	d := lazyjack.Docker{Command: lazyjack.DefaultDockerCommand}
	randBytes := make([]byte, 16)
	rand.Read(randBytes)
	tempName := "test" + hex.EncodeToString(randBytes)
	d.DoCommand("create dummy volume", []string{"volume", "create", tempName})
	defer d.DoCommand("delete dummy volume", []string{"volume", "rm", "-f", tempName})

	actual, err := d.GetVolumeMountPoint(tempName)
	if err != nil {
		t.Fatalf("FAILED: Expected to be able to get volume mount point")
	}
	expected := fmt.Sprintf("/var/lib/docker/volumes/%s/_data", tempName)
	if actual != expected {
		t.Fatalf("FAILED: Getting volume mount point. Expected %q, got %q", expected, actual)
	}
}

func TestFailedGetVolumeMountPoint(t *testing.T) {
	d := lazyjack.Docker{Command: lazyjack.DefaultDockerCommand}

	_, err := d.GetVolumeMountPoint("no-such-volume-exists")
	if err == nil {
		t.Fatalf("FAILED: Expected not to be able to get volume mount point")
	}
	expected := "docker \"volume\" failed for \"Volume inspect\": exit status 1"
	if !strings.HasPrefix(err.Error(), expected) {
		t.Fatalf("FAILED: Expected reason to be  %q, got %q", expected, err.Error())
	}
}
