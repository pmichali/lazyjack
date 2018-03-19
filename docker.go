package lazyjack

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
)

type Docker struct{}

// ResourceExists inspects to see if the resource is known
// to docker.
// Q: Should we check if it is running? If not, remove?
func (d *Docker) ResourceExists(r string) bool {
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

func BuildGetInterfaceArgsForDNS64() []string {
	return []string{"exec", "bind9", "ip", "addr", "list", "eth0"}
}

func BuildV4AddrDelArgsForDNS64(ip string) []string {
	return []string{"exec", "bind9", "ip", "addr", "del", ip, "dev", "eth0"}
}

func BuildAddRouteArgsForDNS64(c *Config) []string {
	return []string{
		"exec", "bind9", "ip", "-6", "route", "add", c.DNS64.CIDR, "via", c.NAT64.ServerIP,
	}
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
		"run", "-d", "--name", "tayga", "--hostname", "tayga", "--label", "lazyjack",
		"--privileged=true", "--ip", c.NAT64.V4MappingIP, "--ip6", c.NAT64.ServerIP,
		"--dns", c.DNS64.RemoteV4Server, "--dns", c.DNS64.ServerIP,
		"--sysctl", "net.ipv6.conf.all.disable_ipv6=0",
		"--sysctl", "net.ipv6.conf.all.forwarding=1",
		"-e", confPrefix, "-e", confV4Addr,
		"--net", SupportNetName, "danehans/tayga:latest",
	}
	return cmdList
}

func BuildCreateNetArgsForSupportNet(cidr, subnet, v4cidr string) []string {
	args := []string{"network", "create", "--ipv6"}
	subnetOption := fmt.Sprintf("--subnet=\"%s\"", cidr)
	v4SubnetOption := fmt.Sprintf("--subnet=%s", v4cidr)
	gw := fmt.Sprintf("--gateway=\"%s1\"", subnet)
	args = append(args, subnetOption, v4SubnetOption, gw, SupportNetName)
	return args
}

func BuildDeleteNetArgsFor(name string) []string {
	return []string{"network", "rm", name}
}

func (d *Docker) DeleteNetwork(name string) error {
	args := BuildDeleteNetArgsFor(name)
	_, err := DoCommand("SupportNetName", args)
	return err
}
