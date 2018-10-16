package lazyjack

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
)

// CreateCertKeyArea creates the area used to hold certificates used
// by KubeAdm.
func CreateCertKeyArea(base string) error {
	area := filepath.Join(base, CertArea)
	err := os.RemoveAll(area)
	if err != nil {
		return fmt.Errorf("unable to clear out certificate area: %v", err)
	}
	err = os.MkdirAll(area, 0700)
	if err != nil {
		return fmt.Errorf("unable to create area for certificates (%s): %v", area, err)
	}
	glog.V(1).Infof("Created area for certificates")
	return nil
}

// BuildArgsForCAKey constructs arguments for command to build CA keys.
func BuildArgsForCAKey(base string) []string {
	return []string{"genrsa", "-out", filepath.Join(base, CertArea, "ca.key"), "2048"}
}

// CreateKeyForCA creates the CA key and stores it in a file.
func CreateKeyForCA(base string) error {
	glog.V(1).Infof("Creating CA key")
	args := BuildArgsForCAKey(base)
	_, err := DoExecCommand("openssl", args)
	if err != nil {
		return fmt.Errorf("unable to create CA key: %v", err)
	}
	glog.Infof("Created CA key")
	return nil
}

// BuildArgsForCACert constructs arguments for command to build CA certificate.
func BuildArgsForCACert(mgmtPrefix string, id int, base string) []string {
	return []string{
		"req", "-x509", "-new", "-nodes",
		"-key", filepath.Join(base, CertArea, "ca.key"),
		"-subj", fmt.Sprintf("/CN=%s%d", mgmtPrefix, id),
		"-days", "10000",
		"-out", filepath.Join(base, CertArea, "ca.crt"),
	}
}

// CreateCertificateForCA creates CA certificate and stores in a file.
func CreateCertificateForCA(mgmtPrefix string, id int, base string) error {
	glog.V(1).Infof("Creating CA certificate")
	args := BuildArgsForCACert(mgmtPrefix, id, base)
	_, err := DoExecCommand("openssl", args)
	if err != nil {
		return fmt.Errorf("unable to create CA certificate: %v", err)
	}
	glog.Info("Created CA certificate")
	return nil
}

// BuildArgsForX509Cert builds args for command to create X.509 certificate.
func BuildArgsForX509Cert(base string) []string {
	return []string{
		"x509", "-pubkey",
		"-in", filepath.Join(base, CertArea, "ca.crt"),
	}

}

// CreateX509CertForCA creates X.509 certificate and stores in a file.
func CreateX509CertForCA(base string) error {
	glog.V(4).Infof("Building CA X509 certificate")
	args := BuildArgsForX509Cert(base)
	output, err := DoExecCommand("openssl", args)
	if err != nil || len(output) == 0 {
		return fmt.Errorf("unable to create X509 cert: %v", err)
	}
	err = ioutil.WriteFile(filepath.Join(base, CertArea, "ca.x509"), []byte(output), 0644)
	if err != nil {
		return fmt.Errorf("unable to save X509 cert for CA: %v", err)
	}
	glog.V(1).Infof("Built CA X509 certificate")
	return nil
}

// BuildArgsForRSA builds arguments for command to create RSA key.
func BuildArgsForRSA(base string) []string {
	return []string{
		"rsa", "-pubin",
		"-in", filepath.Join(base, CertArea, "ca.x509"),
		"-outform", "der",
		"-out", filepath.Join(base, CertArea, "ca.rsa"),
	}
}

// CreateRSAForCA creates RSA key and stores in a file.
func CreateRSAForCA(base string) error {
	glog.V(4).Infof("Building RSA key for CA")
	args := BuildArgsForRSA(base)
	_, err := DoExecCommand("openssl", args)
	if err != nil {
		return fmt.Errorf("unable to create RSA key for CA: %v", err)
	}
	glog.V(1).Infof("Built RSA key for CA")
	return nil
}

// BuildArgsForCADigest builds arguments for comamnd to create CA digest.
func BuildArgsForCADigest(base string) []string {
	return []string{
		"dgst", "-sha256", "-hex",
		filepath.Join(base, CertArea, "ca.rsa"),
	}
}

// ExtractDigest parses the digest, extracting the hash and validating it.
func ExtractDigest(input string) (string, error) {
	glog.V(4).Infof("Parsing digest info %q", input)
	parts := strings.Split(input, " ")
	if len(parts) != 2 {
		return "", fmt.Errorf("unable to parse digest info for CA key")
	}
	hash := strings.TrimSpace(parts[1])
	err := ValidateTokenCertHash(hash, true)
	if err != nil {
		return "", err
	}
	glog.V(1).Infof("Built digest for CA (%s)", hash)
	return hash, nil
}

// CreateDigestForCA creates the CA digest and extracts the hash for it.
func CreateDigestForCA(base string) (string, error) {
	glog.V(4).Infof("Building digest for CA")
	args := BuildArgsForCADigest(base)
	output, err := DoExecCommand("openssl", args)
	if err != nil {
		return "", fmt.Errorf("unable to create CA digest: %v", err)
	}
	return ExtractDigest(output)
}

// ExtractToken extracts the access token and validates it, returning the value.
func ExtractToken(input string) (string, error) {
	glog.V(4).Infof("Parsing token %q", input)
	token := strings.TrimSpace(input)
	err := ValidateToken(token, false)
	if err != nil {
		return "", fmt.Errorf("internal error, token is malformed: %v", err)
	}
	glog.V(1).Infof("Created shared token (%s)", token)
	return token, nil
}

// CreateToken creates the shared token and extracts the value.
func CreateToken() (string, error) {
	glog.V(4).Infof("Creating shared token")
	args := []string{"token", "generate"}
	token, err := DoExecCommand("kubeadm", args)
	if err != nil {
		return "", fmt.Errorf("unable to create shared token: %v", err)
	}
	return ExtractToken(token)
}

// UpdateConfigYAMLContents will parse through the provided config file contents
// and add the token and token certificate hash entries. Old values, if present,
// will be removed. The new fields will be placed inside of the general section.
func UpdateConfigYAMLContents(contents []byte, file, token, hash string) []byte {
	glog.V(4).Infof("Updating %s contents", file)
	lines := bytes.Split(bytes.TrimRight(contents, "\n"), []byte("\n"))
	var output bytes.Buffer
	notHandled := true
	for _, line := range lines {
		if bytes.HasPrefix(bytes.TrimLeft(line, " "), []byte("token:")) {
			continue
		}
		if bytes.HasPrefix(bytes.TrimLeft(line, " "), []byte("token-cert-hash:")) {
			continue
		}
		if bytes.HasPrefix(line, []byte("general:")) {
			output.WriteString(fmt.Sprintf("general:\n"))
			output.WriteString(fmt.Sprintf("    token: %q\n", token))
			output.WriteString(fmt.Sprintf("    token-cert-hash: %q\n", hash))
			notHandled = false
			continue
		}
		output.WriteString(fmt.Sprintf("%s\n", line))
	}
	// Should have general section, so that this is not required, but being rigorous
	if notHandled {
		output.WriteString(fmt.Sprintf("general:\n"))
		output.WriteString(fmt.Sprintf("    token: %q\n", token))
		output.WriteString(fmt.Sprintf("    token-cert-hash: %q\n", hash))
	}
	return output.Bytes()
}

// OpenPermissions helper makes the directory read/write.
func OpenPermissions(name string) error {
	err := os.Chmod(name, 0777)
	if err != nil {
		return fmt.Errorf("unable to open permissions on %q: %v", name, err)
	}
	return nil
}

// UpdateConfigYAML adds the access token and hash to the configuration YAML
// file, replacing any existing entries.
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

// Initialize performs steps for the "init" operation, creating
// certificate, key, token, and hash, and then updates the configuration
// YAML file with the token and hash, so that KubeAdm operations can be
// performed.
func Initialize(name string, c *Config, configFile string) error {
	node := c.Topology[name]

	if !node.IsMaster {
		return nil
	}
	glog.Infof("Initializing master node %q", name)
	base := c.General.WorkArea
	err := CreateCertKeyArea(base)
	if err != nil {
		return err
	}
	err = CreateKeyForCA(base)
	if err != nil {
		return err
	}
	err = CreateCertificateForCA(c.Mgmt.Prefix, node.ID, base)
	if err != nil {
		return err
	}
	err = CreateX509CertForCA(base)
	if err != nil {
		return err
	}
	err = CreateRSAForCA(base)
	if err != nil {
		return err
	}
	hash, err := CreateDigestForCA(base)
	if err != nil {
		return err
	}
	token, err := CreateToken()
	if err != nil {
		return err
	}
	err = UpdateConfigYAML(configFile, token, hash)
	if err != nil {
		return err
	}

	glog.Infof("Node %q initialized", name)
	return nil
}
