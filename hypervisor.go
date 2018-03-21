package lazyjack

type Hypervisor interface {
	ResourceExists(string, bool) bool
	DeleteContainer(string) error
	RunContainer(string, []string) error
	//	NetworkCreate() error
	DeleteNetwork(string) error
	GetInterfaceConfig(string, string) (string, error)
	DeleteV4Address(string, string) error
	AddV6Route(string, string, string) error
}
