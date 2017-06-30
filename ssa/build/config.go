package build

import (
	"fmt"
	"go/build"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/nickng/gospal/ssa"
	"golang.org/x/tools/go/loader"
	gossa "golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// srcReader is a wrapper for source code which can be read through a NewReader.
type srcReader interface {
	NewReader() io.Reader
}

type Configurer interface {
	Builder
	Default() Configurer
	AddBadPkg(pkg, reason string) Configurer
	WithBuildLog(l io.Writer, flags int) Configurer
	WithPtaLog(l io.Writer, flags int) Configurer
}

// Config represents a build configuration.
type Config struct {
	badPkgs map[string]string

	bldLog    io.Writer // Build log.
	bldLFlags int       // Build log flags.
	ptaLog    io.Writer // Pointer analysis log.
	ptaLFlags int       // Pointer analysis log flags.

	src srcReader // src points to the program source.
}

func newConfig(src srcReader) *Config {
	return &Config{
		badPkgs:   make(map[string]string),
		bldLog:    ioutil.Discard,
		bldLFlags: log.LstdFlags,
		ptaLog:    ioutil.Discard,
		ptaLFlags: log.LstdFlags,
		src:       src,
	}
}

// WithBuildLog adds build log to config.
func (c *Config) WithBuildLog(l io.Writer, flags int) Configurer {
	c.bldLog = l
	c.bldLFlags = flags
	return c
}

// WithPtaLog adds pointer analysis log to config.
func (c *Config) WithPtaLog(l io.Writer, flags int) Configurer {
	//c := b.(*Config)
	c.ptaLog = l
	c.ptaLFlags = flags
	return c
}

// AddBadPkg marks a package 'bad' to avoid loading.
func (c *Config) AddBadPkg(pkg, reason string) Configurer {
	//c := b.(*Config)
	c.badPkgs[pkg] = reason
	return c
}

func (c *Config) Build() (*ssa.Info, error) {
	var lconf = loader.Config{Build: &build.Default}
	bldLog := log.New(c.bldLog, "ssabuild: ", c.bldLFlags)

	switch src := c.src.(type) {
	case *FileSrc:
		args, err := lconf.FromArgs(src.Files, false /* No tests */)
		if err != nil {
			return nil, err
		}
		if len(args) > 0 {
			return nil, fmt.Errorf("surplus arguments: %q", args)
		}
	default:
		os.Chdir(os.TempDir())
		parsed, err := lconf.ParseFile("tmp", src.NewReader())
		if err != nil {
			return nil, err
		}
		lconf.CreateFromFiles("", parsed)
	}

	// Load, parse and type-check program
	lprog, err := lconf.Load()
	if err != nil {
		return nil, err
	}
	bldLog.Print("Program loaded and type checked")

	prog := ssautil.CreateProgram(lprog, gossa.GlobalDebug|gossa.BareInits)

	var ignoredPkgs []string
	if len(c.badPkgs) == 0 {
		prog.Build()
	} else {
		for _, info := range lprog.AllPackages {
			if reason, badPkg := c.badPkgs[info.Pkg.Name()]; badPkg {
				bldLog.Printf("Skip package: %s (%s)", info.Pkg.Name(), reason)
				ignoredPkgs = append(ignoredPkgs, info.Pkg.Name())
			} else {
				prog.Package(info.Pkg).Build()
			}
		}
	}

	return &ssa.Info{
		IgnoredPkgs: ignoredPkgs,
		FSet:        lprog.Fset,
		Prog:        prog,
		LProg:       lprog,
		BldLog:      c.bldLog,
		PtaLog:      c.ptaLog,
	}, nil
}

// Default returns a default configuration for static analysis.
func (c *Config) Default() Configurer {
	return c.
		AddBadPkg("reflect", "Reflection is not supported").
		AddBadPkg("runtime", "Runtime is ignored for static analysis")
}
