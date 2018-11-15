package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/pmichali/lazyjack"

	"github.com/golang/glog"
)

const (
	Version = "1.3.4"
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
	var err error

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] {init|prepare|up|down|clean|version}\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}

	thisHost, err := os.Hostname()
	if err != nil {
		thisHost = "" // Hopefully user can specify
	}
	var configFile = flag.String("config", "config.yaml", "Configurations for lazyjack")
	var host = flag.String("host", thisHost, "Name of (this) host to apply command")

	InitLogs()
	defer FlushLogs()

	flag.Parse()

	command, err := lazyjack.ValidateCommand(flag.Arg(0))
	if err != nil {
		fmt.Printf("ERROR: %s\n\n", err.Error())
		flag.Usage()
		os.Exit(1)
	}

	if command == "version" {
		fmt.Printf("Version: %s\n", Version)
		os.Exit(0)
	} else {
		glog.Infof("Version %s", Version)
	}
	cf, err := lazyjack.OpenConfigFile(*configFile)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		os.Exit(1)
	}
	config, err := lazyjack.LoadConfig(cf)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		os.Exit(1)
	}
	ignoreMissing := (command == "init")
	err = lazyjack.ValidateConfigContents(config, ignoreMissing)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		os.Exit(1)
	}
	err = lazyjack.ValidateHost(*host, config)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		os.Exit(1)
	}

	glog.V(1).Infof("Command %q on host %q", command, *host)

	switch command {
	case "init":
		err = lazyjack.Initialize(*host, config, *configFile)
		if err != nil {
			glog.Errorf(err.Error())
			os.Exit(1)
		}
	case "prepare":
		err = lazyjack.Prepare(*host, config)
		if err != nil {
			glog.Errorf(err.Error())
			os.Exit(1)
		}
	case "up":
		lazyjack.BringUp(*host, config)
	case "down":
		lazyjack.TearDown(*host, config)
	case "clean":
		err := lazyjack.Cleanup(*host, config)
		if err != nil {
			glog.Warning(err.Error())
		}
	default:
		fmt.Printf("Unknown command %q\n", command)
		os.Exit(1)
	}
	glog.V(4).Info("Command completed")
}
