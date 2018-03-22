package lazyjack

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
)

type Docker struct{}

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

func DoCommand(name string, args []string) (string, error) {
	glog.V(4).Infof("Invoking: docker %s", strings.Join(args, " "))
	cmd := args[0]
	c := exec.Command("docker", args...)
	output, err := c.Output()
	if err != nil {
		return "", fmt.Errorf("Docker %q failed for %q: %s (%s)", cmd, name, err.Error(), output)
	}
	glog.V(4).Infof("Docker %q successful for %q", cmd, name)
	return string(output), nil
}

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

func BuildGetInterfaceArgs(container, ifName string) []string {
	return []string{"exec", container, "ip", "addr", "list", ifName}
}

func (d *Docker) GetInterfaceConfig(name, ifName string) (string, error) {
	args := BuildGetInterfaceArgs(name, ifName)
	return DoCommand("Get I/F config", args)
}

func BuildV4AddrDelArgs(container, ip string) []string {
	return []string{"exec", container, "ip", "addr", "del", ip, "dev", "eth0"}
}

func (d *Docker) DeleteV4Address(container, ip string) error {
	args := BuildV4AddrDelArgs(container, ip)
	_, err := DoCommand("Delete IPv4 addr", args)
	return err
}

func BuildAddRouteArgs(container, dest, via string) []string {
	return []string{
		"exec", container, "ip", "-6", "route", "add", dest, "via", via,
	}
}

func (d *Docker) AddV6Route(container, dest, via string) error {
	args := BuildAddRouteArgs(container, dest, via)
	_, err := DoCommand("Add IPv6 route", args)
	return err
}

func (d *Docker) DeleteContainer(name string) error {
	args := []string{"rm", "-f", name}
	_, err := DoCommand(name, args)
	return err
}

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

func (d *Docker) RunContainer(name string, args []string) error {
	_, err := DoCommand(name, args)
	return err
}

func BuildCreateNetArgsFor(name, cidr, v4cidr, gw string) []string {
	args := []string{"network", "create", "--ipv6"}
	subnetOption := fmt.Sprintf("--subnet=\"%s\"", cidr)
	v4SubnetOption := fmt.Sprintf("--subnet=%s", v4cidr)
	gwOption := fmt.Sprintf("--gateway=\"%s1\"", gw)
	args = append(args, subnetOption, v4SubnetOption, gwOption, name)
	return args
}

func (d *Docker) CreateNetwork(name, cidr, v4cidr, gw string) error {
	args := BuildCreateNetArgsFor(name, cidr, v4cidr, gw)
	_, err := DoCommand(name, args)
	return err
}

func BuildDeleteNetArgsFor(name string) []string {
	return []string{"network", "rm", name}
}

func (d *Docker) DeleteNetwork(name string) error {
	args := BuildDeleteNetArgsFor(name)
	_, err := DoCommand("SupportNetName", args)
	return err
}
