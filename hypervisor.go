package lazyjack

type Hypervisor interface {
	ResourceExists(string) bool
	DeleteContainer(string) error
	//	RunContainer() error
	//	NetworkCreate() error
	DeleteNetwork(string) error
	//	InterfaceList()
	//	InterfaceDelete() error
	//	RouteAdd() error
}
