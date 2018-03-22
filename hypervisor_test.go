package lazyjack_test

import "fmt"

type MockHypervisor struct {
	simNotExists           bool
	simDeleteNetFail       bool
	simDeleteContainerFail bool
	simNotRunning          bool
	simRunFailed           bool
	simInterfaceGetFail    bool
	simNoV4Interface       bool
	simDeleteInterfaceFail bool
	simAddRouteFail        bool
	simRouteExists         bool
	simCreateNetFail       bool
}

func (mh *MockHypervisor) ResourceExists(r string, requireRunning bool) bool {
	if mh.simNotExists {
		return false
	}
	if mh.simNotRunning {
		return false
	}
	return true
}

func (mh *MockHypervisor) DeleteNetwork(string) error {
	if mh.simDeleteNetFail {
		return fmt.Errorf("Mock fail delete of network")
	}
	return nil
}

func (mh *MockHypervisor) DeleteContainer(name string) error {
	if mh.simDeleteContainerFail {
		return fmt.Errorf("Mock fail delete of container")
	}
	return nil
}

func (mh *MockHypervisor) RunContainer(name string, args []string) error {
	if mh.simRunFailed {
		return fmt.Errorf("Mock fail to run container")
	}
	return nil
}

func (mh *MockHypervisor) GetInterfaceConfig(name, ifName string) (string, error) {
	if mh.simInterfaceGetFail {
		return "", fmt.Errorf("Mock fail getting interface info")
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
		return fmt.Errorf("Mock fail delete of IP")
	}
	return nil
}

func (mh *MockHypervisor) AddV6Route(container, dest, via string) error {
	if mh.simAddRouteFail {
		return fmt.Errorf("Mock fail add route")
	}
	if mh.simRouteExists {
		return fmt.Errorf("file exists")
	}
	return nil
}

func (mh *MockHypervisor) CreateNetwork(name, cidr, v4cidr, gw string) error {
	if mh.simCreateNetFail {
		return fmt.Errorf("Mock fail create of network")
	}
	return nil
}
