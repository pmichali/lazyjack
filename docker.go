package orca

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/golang/glog"
)

// Q: Should we check if it is running? If not, remove?
func ResourceExists(r string) bool {
	_, err := DoCommand(r, []string{"inspect", r})
	if err != nil {
		glog.V(4).Infof("No %q resource", r)
		return false
	}
	glog.V(4).Infof("Resource %q exists", r)
	return true
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
	volumeMap := fmt.Sprintf("%s:/etc/bind/named.conf", DNS64NamedConf)
	cmdList := []string{
		"run", "-d", "--name", "bind9", "--hostname", "bind9", "--label", "orca",
		"--privileged=true", "--ip6", c.DNS64.ServerIP, "--dns", c.DNS64.ServerIP,
		"--sysctl", "net.ipv6.conf.all.disable_ipv6=0",
		"--sysctl", "net.ipv6.conf.all.forwarding=1",
		"-v", volumeMap, "--net", SupportNetName, "resystit/bind9:latest",
	}
	return cmdList
}

func BuildGetInterfaceArgsForDNS64() []string {
	return []string{"exec", "bind9", "ip", "addr", "list", "eth0"}
}

func BuildV4AddrDelArgsForDNS64(ip string) []string {
	return []string{"exec", "bind9", "ip", "addr", "del", ip, "dev", "eth0"}
}

func BuildAddRouteArgsForDNS64(c *Config) []string {
	prefixCIDR := fmt.Sprintf("%s/%d", c.DNS64.Prefix, c.DNS64.PrefixSize)
	return []string{
		"exec", "bind9", "ip", "-6", "route", "add", prefixCIDR, "via", c.NAT64.ServerIP,
	}
}

func RemoveDNS64Container() error {
	if ResourceExists("bind9") {
		args := []string{"rm", "-f", "bind9"}
		_, err := DoCommand("bind9", args)
		return err
	}
	return nil
}

func BuildRunArgsForNAT64(c *Config) []string {
	confPrefix := fmt.Sprintf("TAYGA_CONF_PREFIX=%s/%d", c.DNS64.Prefix, c.DNS64.PrefixSize)
	confV4Addr := fmt.Sprintf("TAYGA_CONF_IPV4_ADDR=%s", c.NAT64.V4MappingIP)
	cmdList := []string{
		"run", "-d", "--name", "tayga", "--hostname", "tayga", "--label", "orca",
		"--privileged=true", "--ip", c.NAT64.V4MappingIP, "--ip6", c.NAT64.ServerIP,
		"--dns", c.DNS64.RemoteV4Server, "--dns", c.DNS64.ServerIP,
		"--sysctl", "net.ipv6.conf.all.disable_ipv6=0",
		"--sysctl", "net.ipv6.conf.all.forwarding=1",
		"-e", confPrefix, "-e", confV4Addr,
		"--net", SupportNetName, "danehans/tayga:latest",
	}
	return cmdList
}

func RemoveNAT64Container() error {
	if ResourceExists("tayga") {
		args := []string{"rm", "-f", "tayga"}
		_, err := DoCommand("tayga", args)
		return err
	}
	return nil
}

func BuildCreateNetArgsForSupportNet(subnet string, subnetSize int, v4cidr string) []string {
	args := []string{"network", "create", "--ipv6"}
	subnetOption := fmt.Sprintf("--subnet=\"%s/%d\"", subnet, subnetSize)
	v4SubnetOption := fmt.Sprintf("--subnet=%s", v4cidr)
	gw := fmt.Sprintf("--gateway=\"%s1\"", subnet)
	args = append(args, subnetOption, v4SubnetOption, gw, SupportNetName)
	return args
}

func BuildDeleteNetArgsForSupportNet() []string {
	return []string{"network", "rm", SupportNetName}
}
