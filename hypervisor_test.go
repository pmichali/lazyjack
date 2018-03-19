package lazyjack_test

import "fmt"

type MockHypervisor struct {
	simNotExists           bool
	simDeleteNetFail       bool
	simDeleteContainerFail bool
}

func (mh *MockHypervisor) ResourceExists(string) bool {
	if mh.simNotExists {
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
