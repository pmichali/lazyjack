package lazyjack

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/golang/glog"
)

func CreateCertKeyArea() error {
	err := os.RemoveAll(CertArea)
	if err != nil {
		return fmt.Errorf("Unable to clear out certificate area: %s", err.Error())
	}
	err = os.MkdirAll(CertArea, 0700)
	if err != nil {
		return fmt.Errorf("Unable to create area for certificates (%s): %s", CertArea, err.Error())
	}
	glog.V(1).Infof("Created area for certificates")
	return nil
}

func BuildArgsForCAKey() []string {
	return []string{"genrsa", "-out", fmt.Sprintf("%s/%s", CertArea, "ca.key"), "2048"}
}

func CreateKeyForCA() error {
	glog.V(1).Infof("Creating CA key")
	args := BuildArgsForCAKey()
	_, err := DoExecCommand("openssl", args)
	if err != nil {
		return fmt.Errorf("Unable to create CA key: %s", err.Error())
	}
	glog.Infof("Created CA key")
	return nil
}

func BuildArgsForCACert(n *Node, c *Config) []string {
	return []string{
		"req", "-x509", "-new", "-nodes",
		"-key", fmt.Sprintf("%s/ca.key", CertArea),
		"-subj", fmt.Sprintf("/CN=%s%d", c.Mgmt.Prefix, n.ID),
		"-days", "10000",
		"-out", fmt.Sprintf("%s/ca.crt", CertArea),
	}
}

func CreateCertificateForCA(n *Node, c *Config) error {
	glog.V(1).Infof("Creating CA certificate")
	args := BuildArgsForCACert(n, c)
	_, err := DoExecCommand("openssl", args)
	if err != nil {
		return fmt.Errorf("Unable to create CA certificate: %s", err.Error())
	}
	glog.Infof("Created CA certificate")
	return nil
}

func BuildArgsForX509Cert() []string {
	return []string{
		"x509", "-pubkey",
		"-in", fmt.Sprintf("%s/ca.crt", CertArea),
	}

}

func CreateX509CertForCA() error {
	glog.V(4).Infof("Building CA X509 certificate")
	args := BuildArgsForX509Cert()
	output, err := DoExecCommand("openssl", args)
	if err != nil || len(output) == 0 {
		return fmt.Errorf("Unable to create X509 cert: %s", err.Error())
	}
	err = ioutil.WriteFile(fmt.Sprintf("%s/ca.x509", CertArea), []byte(output), 0644)
	if err != nil {
		return fmt.Errorf("Unable to save X509 cert for CA", err.Error())
	}
	glog.V(1).Infof("Built CA X509 certificate")
	return nil
}

func BuildArgsForRSA() []string {
	return []string{
		"rsa", "-pubin",
		"-in", fmt.Sprintf("%s/ca.x509", CertArea),
		"-outform", "der",
		"-out", fmt.Sprintf("%s/ca.rsa", CertArea),
	}
}

func CreateRSAForCA() error {
	glog.V(4).Infof("Building RSA key for CA")
	args := BuildArgsForRSA()
	_, err := DoExecCommand("openssl", args)
	if err != nil {
		return fmt.Errorf("Unable to create RSA key for CA: %s", err.Error())
	}
	glog.V(1).Infof("Built RSA key for CA")
	return nil
}

func BuildArgsForCADigest() []string {
	return []string{
		"dgst", "-sha256", "-hex",
		fmt.Sprintf("%s/ca.rsa", CertArea),
	}
}

func CreateDigestForCA() (string, error) {
	glog.V(4).Infof("Building digest for CA")
	args := BuildArgsForCADigest()
	output, err := DoExecCommand("openssl", args)
	if err != nil {
		return "", fmt.Errorf("Unable to create CA digest: %s", err.Error())
	}
	parts := strings.Split(output, " ")
	if len(parts) != 2 {
		return "", fmt.Errorf("Unable to parse digest info for CA key")
	}
	hash := strings.TrimSpace(parts[1])
	err = ValidateTokenCertHash(hash, true)
	if err != nil {
		return "", err
	}
	glog.V(1).Infof("Built digest for CA (%s)", hash)
	return hash, nil
}

func CreateCertficateHashForCA() (string, error) {
	err := CreateX509CertForCA()
	if err != nil {
		return "", err
	}
	err = CreateRSAForCA()
	if err != nil {
		return "", err
	}
	return CreateDigestForCA()
}

func CreateToken() (string, error) {
	glog.V(4).Infof("Creating shared token")
	args := []string{"token", "generate"}
	token, err := DoExecCommand("kubeadm", args)
	if err != nil {
		return "", fmt.Errorf("Unable to create shared token: %s", err.Error())
	}
	token = strings.TrimSpace(token)
	err = ValidateToken(token, false)
	if err != nil {
		return "", fmt.Errorf("Internal error, token is malformed: %s", err.Error())
	}
	glog.V(1).Infof("Created shared token (%s)", token)
	return token, nil
}

func UpdateConfigYAMLContents(contents []byte, file, token, hash string) []byte {
	glog.V(4).Infof("Updating %s contents", file)
	lines := bytes.Split(bytes.TrimRight(contents, "\n"), []byte("\n"))
	var output bytes.Buffer
	sawPlugin := false
	notHandled := true
	for _, line := range lines {
		if bytes.HasPrefix(line, []byte("plugin:")) {
			sawPlugin = true
		} else if sawPlugin && notHandled {
			output.WriteString(fmt.Sprintf("token: %q\n", token))
			output.WriteString(fmt.Sprintf("token-cert-hash: %q\n", hash))
			notHandled = false
		}
		if bytes.HasPrefix(line, []byte("token:")) {
			continue
		}
		if bytes.HasPrefix(line, []byte("token-cert-hash:")) {
			continue
		}
		output.WriteString(fmt.Sprintf("%s\n", line))
	}
	return output.Bytes()
}

func OpenPermissions(name string) error {
	err := os.Chmod(name, 0777)
	if err != nil {
		return fmt.Errorf("Unable to open permissions on %q: %s", name, err.Error())
	}
	return nil
}

func UpdateConfigYAML(file, token, hash string) error {
	glog.V(1).Infof("Updating %s file", file)
	contents, err := GetFileContents(file)
	if err != nil {
		return err
	}
	contents = UpdateConfigYAMLContents(contents, file, token, hash)
	backup := fmt.Sprintf("%s.bak", file)
	err = SaveFileContents(contents, file, backup)
	if err != nil {
		return err
	}
	err = OpenPermissions(file)
	if err != nil {
		return err
	}
	err = OpenPermissions(backup)
	if err != nil {
		return err
	}
	glog.Infof("Updated %s file", file)
	return nil
}

func Initialize(name string, c *Config, configFile string) {
	node := c.Topology[name]

	if !node.IsMaster {
		return
	}
	glog.Infof("Initializing master node %q", name)
	err := CreateCertKeyArea()
	if err != nil {
		glog.Errorf(err.Error())
		os.Exit(1)
	}

	err = CreateKeyForCA()
	if err != nil {
		glog.Errorf(err.Error())
		os.Exit(1)
	}
	err = CreateCertificateForCA(&node, c)
	if err != nil {
		glog.Errorf(err.Error())
		os.Exit(1)
	}
	hash, err := CreateCertficateHashForCA()
	if err != nil {
		glog.Errorf(err.Error())
		os.Exit(1)
	}
	token, err := CreateToken()
	if err != nil {
		glog.Errorf(err.Error())
		os.Exit(1)
	}
	err = UpdateConfigYAML(configFile, token, hash)
	if err != nil {
		glog.Errorf(err.Error())
		os.Exit(1)
	}

	glog.Infof("Node %q initialized", name)
}
