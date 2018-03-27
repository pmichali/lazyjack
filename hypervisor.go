package lazyjack

// Hypervisor interface indicates the general API for hypervisor operations.
type Hypervisor interface {
	ResourceState(r string) string
	DeleteContainer(string) error
	RunContainer(string, []string) error
	CreateNetwork(string, string, string, string) error
	DeleteNetwork(string) error
	GetInterfaceConfig(string, string) (string, error)
	DeleteV4Address(string, string) error
	AddV6Route(string, string, string) error
}
