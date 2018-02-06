package main

import (
	"flag"
	"fmt"
	"github.com/golang/glog"
	"log"
	"orca"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	flag.Set("logtostderr", "true")
}

// GlogWriter serves as a bridge between the standard log package and the glog package.
type GlogWriter struct{}

// Write implements the io.Writer interface.
func (writer GlogWriter) Write(data []byte) (n int, err error) {
	glog.Info(string(data))
	return len(data), nil
}

// InitLogs initializes logs the way we want for kubernetes.
func InitLogs() {
	log.SetOutput(GlogWriter{})
	log.SetFlags(0)
}

// FlushLogs flushes logs immediately.
func FlushLogs() {
	glog.Flush()
}

func validateCommand(command string) string {
	if command == "" {
		fmt.Printf("ERROR: Missing command.\n\n")
		flag.Usage()
		os.Exit(1)
	}
	validCommands := []string{"prepare", "up", "down", "clean"}
	for _, c := range validCommands {
		if strings.EqualFold(c, command) {
			return c
		}
	}
	fmt.Printf("ERROR: Unknown command %q.\n\n", command)
	flag.Usage()
	os.Exit(1)
	return ""
}

func validateHost(host string, config *orca.Config) *orca.Node {
	nodeInfo, ok := config.Topology[host]
	if !ok {
		glog.Fatalf("Unable to find info for host %q in config file", host)
	}
	return &nodeInfo
}

func validateAndLoadConfig(configFile string) *orca.Config {
	glog.V(1).Infof("Using config %q", configFile)

	cf, err := os.Open(configFile)
	if err != nil {
		glog.Fatalf("Unable to open config file %q: %s", configFile, err.Error())
	}
	defer cf.Close()

	config, err := orca.ParseConfig(cf)
	if err != nil {
		glog.Fatal(err)
	}

	return config
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] {prepare|up|down|clean}\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}

	thisHost, err := os.Hostname()
	if err != nil {
		thisHost = "" // Hopefully user can specify
	}
	var configFile = flag.String("config", "config.yaml", "Configurations for orca")
	var host = flag.String("host", thisHost, "Name of (this) host to apply command")

	InitLogs()
	defer FlushLogs()

	flag.Parse()

	command := validateCommand(flag.Arg(0))
	config := validateAndLoadConfig(*configFile)
	hostInfo := validateHost(*host, config)

	glog.V(1).Infof("Command %q on host %q", command, *host)

	fmt.Printf("Host info %+v\n", hostInfo)
	
	switch command {
	case "prepare":
		fmt.Printf("TODO %q on %q\n", command, *host)
	case "up":
		fmt.Printf("TODO %q on %q\n", command, *host)
	case "down":
		fmt.Printf("TODO %q on %q\n", command, *host)
	case "clean":
		fmt.Printf("TODO %q on %q\n", command, *host)
	default:
		fmt.Printf("Unknown command %q\n", command)
		os.Exit(1)
	}
	glog.V(4).Info("Done")
}
