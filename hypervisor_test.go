package lazyjack_test

import (
	"fmt"

	"github.com/pmichali/lazyjack"
)

type MockHypervisor struct {
	simNotExists           bool
	simRunning             bool
	simDeleteNetFail       bool
	simDeleteContainerFail bool
	simRunFailed           bool
	simInterfaceGetFail    bool
	simNoV4Interface       bool
	simDeleteInterfaceFail bool
	simAddRouteFail        bool
	simRouteExists         bool
	simCreateNetFail       bool
}

func (mh *MockHypervisor) ResourceState(r string) string {
	if mh.simNotExists {
		return lazyjack.ResourceNotPresent
	}
	if mh.simRunning {
		return lazyjack.ResourceRunning
	}
	return lazyjack.ResourceExists
}

func (mh *MockHypervisor) DeleteNetwork(string) error {
	if mh.simDeleteNetFail {
		return fmt.Errorf("mock fail delete of network")
	}
	return nil
}

func (mh *MockHypervisor) DeleteContainer(name string) error {
	if mh.simDeleteContainerFail {
		return fmt.Errorf("mock fail delete of container")
	}
	return nil
}

func (mh *MockHypervisor) RunContainer(name string, args []string) error {
	if mh.simRunFailed {
		return fmt.Errorf("mock fail to run container")
	}
	return nil
}

func (mh *MockHypervisor) GetInterfaceConfig(name, ifName string) (string, error) {
	if mh.simInterfaceGetFail {
		return "", fmt.Errorf("mock fail getting interface info")
	}
	ifConfig := `39139: eth0@if39140: <BROADCAST,MULTICAST,UP,LOWER_UP,M-DOWN> mtu 1500 qdisc noqueue state UP
    link/ether 02:42:ac:12:00:02 brd ff:ff:ff:ff:ff:ff
    inet 172.18.0.2/16 scope global eth0
       valid_lft forever preferred_lft forever
    inet6 fd00:10::100/64 scope global flags 02
       valid_lft forever preferred_lft forever
    inet6 fe80::42:acff:fe12:2/64 scope link
       valid_lft forever preferred_lft forever`
	if mh.simNoV4Interface {
		ifConfig = `39139: eth0@if39140: <BROADCAST,MULTICAST,UP,LOWER_UP,M-DOWN> mtu 1500 qdisc noqueue state UP
    link/ether 02:42:ac:12:00:02 brd ff:ff:ff:ff:ff:ff
    inet6 fd00:10::100/64 scope global flags 02
       valid_lft forever preferred_lft forever
    inet6 fe80::42:acff:fe12:2/64 scope link
       valid_lft forever preferred_lft forever`
	}
	return ifConfig, nil
}

func (mh *MockHypervisor) DeleteV4Address(container, ip string) error {
	if mh.simDeleteInterfaceFail {
		return fmt.Errorf("mock fail delete of IP")
	}
	return nil
}

func (mh *MockHypervisor) AddV6Route(container, dest, via string) error {
	if mh.simAddRouteFail {
		return fmt.Errorf("mock fail add route")
	}
	if mh.simRouteExists {
		return fmt.Errorf("exit status 2")
	}
	return nil
}

func (mh *MockHypervisor) CreateNetwork(name, cidr, v4cidr, gw string) error {
	if mh.simCreateNetFail {
		return fmt.Errorf("mock fail create of network")
	}
	return nil
}
