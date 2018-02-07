package main

import (
	"flag"
	"fmt"
	"github.com/golang/glog"
	"log"
	"orca"
	"os"
	"path/filepath"
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

	command, err := orca.ValidateCommand(flag.Arg(0))
	if err != nil {
		fmt.Printf("ERROR: %s\n\n", err.Error())
		flag.Usage()
		os.Exit(1)
	}
	cf, err := orca.ValidateConfigFile(*configFile)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		os.Exit(1)
	}
	config, err := orca.LoadConfig(cf)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		os.Exit(1)
	}
	err = ValidateConfigContents(config)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		os.Exit(1)
	}
	hostInfo, err := orca.ValidateHost(*host, config)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		os.Exit(1)
	}

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
