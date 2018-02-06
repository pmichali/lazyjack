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
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [hostname]\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}

	var configFile = flag.String("config", "config.yaml", "Configurations for orca")
	InitLogs()
	defer FlushLogs()

	flag.Parse()

	glog.V(1).Infof("Using config %q", *configFile)

	cf, err := os.Open(*configFile)
	if err != nil {
		glog.Fatalf("Unable to open config file %q: %s", *configFile, err.Error())
	}
	defer cf.Close()

	config, err := orca.ParseConfig(cf)
	if err != nil {
		glog.Fatal(err)
	}
	fmt.Printf("Configuration is %+v\n", config)
	glog.V(4).Info("Done")
}
