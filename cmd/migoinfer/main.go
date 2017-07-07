// Command migoinfer is the command line entry point to MiGo type inference.
package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"

	"github.com/nickng/gospal/migoinfer"
	"github.com/nickng/gospal/ssa/build"
)

var (
	logPath   = flag.String("log", "", "Specify analysis log file (use '-' for stderr)")
	showRaw   = flag.Bool("raw", false, "Show raw unfiltered MiGo")
	logWriter = ioutil.Discard
	logFile   = ""
)

func init() {
	flag.Parse()
}

func main() {
	conf := build.FromFiles(flag.Args()).Default()
	switch *logPath {
	case "":
	case "-":
		logWriter = os.Stderr
		conf.WithBuildLog(logWriter, log.LstdFlags)
	default:
		f, err := os.Create(*logPath)
		if err != nil {
			log.Fatalf("Cannot create log %s: %v", *logPath, err)
		}
		defer f.Close()
		conf = conf.WithBuildLog(f, log.LstdFlags)
		logWriter = f
		logFile = f.Name()
	}
	info, err := conf.Build()
	if err != nil {
		log.Fatal("Build failed:", err)
	}
	inferer := migoinfer.New(info, logWriter)
	if logFile != "" {
		inferer.AddLogFiles(logFile)
	}
	inferer.SetOutput(os.Stdout)
	if *showRaw {
		inferer.Raw = true
	}
	inferer.Analyse()
}
