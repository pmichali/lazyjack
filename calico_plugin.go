package lazyjack

import (
	"io"
)

// CalicoPlugin implements the actions needed for the Calico CNI plugin.
type CalicoPlugin struct {
	Config *Config
}

// WriteConfigContents is a no-op for Calico plugin
func (c CalicoPlugin) WriteConfigContents(node *Node, w io.Writer) (err error) {
	return nil
}

// Setup is no-op for Calico plugin (or do we create template file?)
func (c CalicoPlugin) Setup(n *Node) error {
	return nil
}

// Cleanup is a no-op for Calico plugin?
func (c CalicoPlugin) Cleanup(n *Node) error {
	return nil
}

// Start applies YAML files created for the Calico CNI plugin
func (c CalicoPlugin) Start() error {
	return nil
}
