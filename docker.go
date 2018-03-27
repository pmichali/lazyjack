package lazyjack

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
)

// Docker represents a concrete hypervisor implementation.
type Docker struct{}

// ResourceState method obtains the state of the resource, which can be
// not present, existing, or running (for container resources).
func (d *Docker) ResourceState(r string) string {
	output, err := DoCommand(r, []string{"inspect", r})
	if err != nil {
		glog.V(4).Infof("No %q resource", r)
		return ResourceNotPresent
	}
	if strings.Contains(output, "\"Running\": true") {
		glog.V(4).Infof("Resource %q is running", r)
		return ResourceRunning
	}
	glog.V(4).Infof("Resource %q exists", r)
	return ResourceExists
}

// DoCommand performs a docker command, collecting and returning output.
// TODO: Perform in a separate go-routine with a timeout, and abort handling.
func DoCommand(name string, args []string) (string, error) {
	glog.V(4).Infof("Invoking: docker %s", strings.Join(args, " "))
	cmd := args[0]
	c := exec.Command("docker", args...)
	output, err := c.Output()
	if err != nil {
		return "", fmt.Errorf("docker %q failed for %q: %v (%s)", cmd, name, err, output)
	}
	glog.V(4).Infof("Docker %q successful for %q", cmd, name)
	return string(output), nil
}

// BuildRunArgsForDNS64 constructs docker command to start DNS64 container.
func BuildRunArgsForDNS64(c *Config) []string {
	conf := filepath.Join(c.General.WorkArea, DNS64BaseArea, DNS64ConfArea, DNS64NamedConf)
	volumeMap := fmt.Sprintf("%s:/etc/bind/named.conf", conf)
	cmdList := []string{
		"run", "-d", "--name", DNS64Name, "--hostname", DNS64Name, "--label", "lazyjack",
		"--privileged=true", "--ip6", c.DNS64.ServerIP, "--dns", c.DNS64.ServerIP,
		"--sysctl", "net.ipv6.conf.all.disable_ipv6=0",
		"--sysctl", "net.ipv6.conf.all.forwarding=1",
		"-v", volumeMap, "--net", SupportNetName, "resystit/bind9:latest",
	}
	return cmdList
}

// BuildGetInterfaceArgs constructs arguments for obtaining list of IPs
// for an interface.
func BuildGetInterfaceArgs(container, ifName string) []string {
	return []string{"exec", container, "ip", "addr", "list", ifName}
}

// GetInterfaceConfig performs docker command to obtain an interface's
// IP addresses.
func (d *Docker) GetInterfaceConfig(name, ifName string) (string, error) {
	args := BuildGetInterfaceArgs(name, ifName)
	return DoCommand("Get I/F config", args)
}

// BuildV4AddrDelArgs constructs arguments for deleting an IPv4 address
// from and interface.
func BuildV4AddrDelArgs(container, ip string) []string {
	return []string{"exec", container, "ip", "addr", "del", ip, "dev", "eth0"}
}

// DeleteV4Address performs docker command to remove the IPv4 address from
// the container's eth0 interface.
func (d *Docker) DeleteV4Address(container, ip string) error {
	args := BuildV4AddrDelArgs(container, ip)
	_, err := DoCommand("Delete IPv4 addr", args)
	return err
}

// BuildAddRouteArgs constructs arguments for adding an IPv6 route to container.
func BuildAddRouteArgs(container, dest, via string) []string {
	return []string{
		"exec", container, "ip", "-6", "route", "add", dest, "via", via,
	}
}

// AddV6Route performs docker command to add an IPv6 route.
func (d *Docker) AddV6Route(container, dest, via string) error {
	args := BuildAddRouteArgs(container, dest, via)
	_, err := DoCommand("Add IPv6 route", args)
	return err
}

// DeleteContainer performs docker command to remove a container.
func (d *Docker) DeleteContainer(name string) error {
	args := []string{"rm", "-f", name}
	_, err := DoCommand(name, args)
	return err
}

// BuildRunArgsForNAT64 constructs arguments to start a NAT64 container.
func BuildRunArgsForNAT64(c *Config) []string {
	confPrefix := fmt.Sprintf("TAYGA_CONF_PREFIX=%s", c.DNS64.CIDR)
	confV4Addr := fmt.Sprintf("TAYGA_CONF_IPV4_ADDR=%s", c.NAT64.V4MappingIP)
	cmdList := []string{
		"run", "-d", "--name", NAT64Name, "--hostname", NAT64Name, "--label", "lazyjack",
		"--privileged=true", "--ip", c.NAT64.V4MappingIP, "--ip6", c.NAT64.ServerIP,
		"--dns", c.DNS64.RemoteV4Server, "--dns", c.DNS64.ServerIP,
		"--sysctl", "net.ipv6.conf.all.disable_ipv6=0",
		"--sysctl", "net.ipv6.conf.all.forwarding=1",
		"-e", confPrefix, "-e", confV4Addr,
		"--net", SupportNetName, "danehans/tayga:latest",
	}
	return cmdList
}

// RunContainer performs docker command to run a container.
func (d *Docker) RunContainer(name string, args []string) error {
	_, err := DoCommand(name, args)
	return err
}

// BuildCreateNetArgsFor constructs arguments to create a docker network.
func BuildCreateNetArgsFor(name, cidr, v4cidr, gw string) []string {
	args := []string{"network", "create", "--ipv6"}
	subnetOption := fmt.Sprintf("--subnet=\"%s\"", cidr)
	v4SubnetOption := fmt.Sprintf("--subnet=%s", v4cidr)
	gwOption := fmt.Sprintf("--gateway=\"%s1\"", gw)
	args = append(args, subnetOption, v4SubnetOption, gwOption, name)
	return args
}

// CreateNetwork performs docker command to create a network.
func (d *Docker) CreateNetwork(name, cidr, v4cidr, gw string) error {
	args := BuildCreateNetArgsFor(name, cidr, v4cidr, gw)
	_, err := DoCommand(name, args)
	return err
}

// BuildDeleteNetArgsFor constructs arguments to delete a network.
func BuildDeleteNetArgsFor(name string) []string {
	return []string{"network", "rm", name}
}

// DeleteNetwork performs docker command to delete a network.
func (d *Docker) DeleteNetwork(name string) error {
	args := BuildDeleteNetArgsFor(name)
	_, err := DoCommand("SupportNetName", args)
	return err
}
