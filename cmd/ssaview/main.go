// Command ssaview is a SSA printer using standard static analysis options.
//
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/nickng/gospal/ssa/build"
)

const (
	Usage = `ssaview is a tool for printing SSA IR of Go source code.

Usage:

  ssaview [options] file.go [files.go...]

Options:

`
)

var (
	buildlogPath string
	defaultArgs  bool
	outPath      string
	viewFunc     string

	out io.Writer
)

const mainMain = "main.main"

func init() {
	flag.BoolVar(&defaultArgs, "default", true, "Use default SSA build arguments")
	flag.StringVar(&buildlogPath, "log", "", "Specify build log file (use '-' for stdout)")
	flag.StringVar(&outPath, "out", "", "Specify output file (default: stdout)")
	flag.StringVar(&viewFunc, "func", mainMain, `Specify the function to view (format: (import/path).FuncName`)
}

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintf(os.Stderr, Usage)
		flag.PrintDefaults()
		os.Exit(0)
	}

	conf := build.FromFiles(flag.Args()...)
	if defaultArgs {
		conf = conf.Default()
	}

	switch buildlogPath {
	case "":
	case "-":
		conf = conf.WithBuildLog(os.Stdout, log.LstdFlags)
	default:
		f, err := os.Create(buildlogPath)
		if err != nil {
			log.Fatalf("Cannot create log %s: %v", buildlogPath, err)
		}
		defer f.Close()
		conf = conf.WithBuildLog(f, log.LstdFlags)
	}

	switch outPath {
	case "":
		out = os.Stdout
	default:
		f, err := os.Create(outPath)
		if err != nil {
			log.Fatalf("Cannot create output file %s: %v", outPath, err)
		}
		defer f.Close()
		out = f
	}

	info, err := conf.Build()
	if err != nil {
		log.Fatal("Cannot build SSA from files:", err)
	}
	if viewFunc != mainMain {
		if _, err := info.WriteFunc(out, viewFunc); err != nil {
			log.Fatal("Cannot write SSA:", err)
		}
	} else {
		if _, err := info.WriteTo(out); err != nil {
			log.Fatal("Cannot write SSA:", err)
		}
	}
}
