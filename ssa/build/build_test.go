package build_test

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/nickng/gospal/ssa"
	"github.com/nickng/gospal/ssa/build"
	gossa "golang.org/x/tools/go/ssa"
)

var (
	helloProg = `
	package main
	import "fmt"
	func main() {
		fmt.Println("hello")
	}`
	emptyProg = `package main; func main() {}`

	testdir string
)

func init() {
	testdir, _ = os.Getwd() // Save the dir where the test files are, for the runnable examples.
}

// Test loading from files.
func TestBuildFromFiles(t *testing.T) {
	files := []string{"testdata/main.go", "testdata/foo.go", "testdata/bar.go"}
	conf := build.FromFiles(files)
	info, err := conf.Build()
	if err != nil {
		t.Errorf("SSA build failed: %v", err)
	}
	mains, err := ssa.MainPkgs(info.Prog, false)
	if err != nil {
		t.Errorf("cannot find main package: %v", err)
	}
	for _, main := range mains {
		if main.Func("main") == nil {
			t.Errorf("cannot find main.main()")
		}
		if main.Func("foo") == nil {
			t.Errorf("cannot find main.foo()")
		}
		if main.Func("bar") == nil {
			t.Errorf("cannot find main.bar()")
		}
	}
}

// Test loading from string/reader.
func TestBuildFromReader(t *testing.T) {
	conf := build.FromReader(strings.NewReader(helloProg))
	info, err := conf.Build()
	if err != nil {
		t.Errorf("SSA build failed: %v", err)
	}
	mains, err := ssa.MainPkgs(info.Prog, false)
	if err != nil {
		t.Errorf("cannot find main package: %v", err)
	}
	for _, main := range mains {
		if main.Func("main") == nil {
			t.Errorf("cannot find main.main()")
		}
	}
}

func TestWithBuildLog(t *testing.T) {
	buf := new(bytes.Buffer)
	conf := build.FromReader(strings.NewReader(helloProg)).WithBuildLog(buf, log.LstdFlags)
	info, err := conf.Build()
	if err != nil {
		t.Errorf("SSA build failed: %v", err)
	}
	if info.BldLog != buf {
		t.Errorf("Expects build log to propagate to built SSA, but got: %v",
			info.BldLog)
	}
	if !strings.Contains(buf.String(), "Program loaded and type checked") {
		t.Errorf("Build log was set but not written to\nlog contains:\n%s",
			buf.String())
	}
}

func TestWithPtaLog(t *testing.T) {
	conf := build.FromReader(strings.NewReader(helloProg)).WithPtaLog(os.Stdout, log.LstdFlags)
	info, err := conf.Build()
	if err != nil {
		t.Errorf("SSA build failed: %v", err)
	}
	if info.PtaLog != os.Stdout {
		t.Errorf("Expects pta log to propagate to built SSA, but got: %v",
			info.PtaLog)
	}
}

func TestAddBadPkg(t *testing.T) {
	conf := build.FromReader(strings.NewReader(helloProg))
	info, err := conf.Build()
	if err != nil {
		t.Errorf("SSA build failed: %v", err)
	}
	for _, pkg := range info.Prog.AllPackages() {
		if pkg.Pkg.Name() == "fmt" && pkg.Members["Printf"].(*gossa.Function).Blocks == nil {
			t.Errorf("fmt package is built but fmt.Printf funcbody is not in SSA")
		}
	}

	confNoFmt := build.FromReader(strings.NewReader(helloProg)).AddBadPkg("fmt", "Fmt adds many pkg dependencies")
	infoNoFmt, err := confNoFmt.Build()
	if err != nil {
		t.Errorf("SSA build failed: %v", err)
	}
	foundFmt := false
	for _, pkg := range infoNoFmt.IgnoredPkgs {
		if pkg == "fmt" {
			foundFmt = true
		}
	}
	if !foundFmt {
		t.Errorf("Expects fmt to be ignored during build (in config.badPkgs)")
	}

	for _, pkg := range infoNoFmt.Prog.AllPackages() {
		if pkg.Pkg.Name() == "fmt" && pkg.Members["Printf"].(*gossa.Function).Blocks != nil {
			t.Errorf("fmt package is not built but fmt.Printf funcbody is in SSA")
		}
	}
}

func ExampleFromFiles() {
	os.Chdir(testdir)
	files := []string{"testdata/main.go", "testdata/foo.go", "testdata/bar.go"}
	conf := build.FromFiles(files)
	info, err := conf.Build()
	if err != nil {
		log.Fatalf("SSA build failed: %v", err)
	}
	_ = info // Use info here
	// output:
}

func ExampleFromReader() {
	conf := build.FromReader(strings.NewReader("package main; func main() {}"))
	info, err := conf.Build()
	if err != nil {
		log.Fatalf("SSA build failed: %v", err)
	}
	_ = info // Use info here
	// output:
}
