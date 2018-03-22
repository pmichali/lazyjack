package lazyjack

type Hypervisor interface {
	ResourceExists(string, bool) bool
	DeleteContainer(string) error
	RunContainer(string, []string) error
	CreateNetwork(string, string, string, string) error
	DeleteNetwork(string) error
	GetInterfaceConfig(string, string) (string, error)
	DeleteV4Address(string, string) error
	AddV6Route(string, string, string) error
}
