package migoinfer_test

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/nickng/gospal/migoinfer"
	"github.com/nickng/gospal/ssa/build"
)

func init() {
	setTdRoot()
}

const testdata = "./testdata"

var tdRoot string // test data root path.

func setTdRoot() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("cannot get working directory: %v", err)
	}
	tdRoot = path.Join(cwd, testdata)
}

// MiGoExpect is the file containing the expected output of MiGo inference.
const MiGoExpect = "migoinfer.expect"

func TestPrimitives(t *testing.T) {
	tests := []struct {
		name   string
		srcDir string // Input Go source dirs.
	}{
		{"Send", "send"},
		{"Recv", "recv"},
		{"Close", "close"},
		{"Select", "select"},
		{"Select2", "select2"},
		{"Select with Default", "select-default"},
		{"Select with Empty continuations", "select-nocont"},
		{"Closure", "closure-send"},
		{"Complex loop", "loop-complex"},
		{"Channel direction", "chandir"},
		{"Return channel/Set channel in struct", "returnch-setch"},
		{"Structs/Interfaces", "iface"},
		{"Channel chain by overwriting chan vars", "overwrite-chan"},
		{"While-true loop", "whiletrue"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testdir := path.Join(tdRoot, test.srcDir)
			migofile := path.Join(testdir, MiGoExpect)
			migob, err := ioutil.ReadFile(migofile)
			if err != nil {
				t.Errorf("cannot read output file: %v", err)
			}

			files, err := ioutil.ReadDir(testdir)
			if err != nil {
				t.Errorf("cannot read dir: %v", err)
			}
			var filenames []string
			for _, file := range files {
				if path.Ext(file.Name()) == ".go" {
					filenames = append(filenames, path.Join(testdir, file.Name()))
				}
			}
			if len(filenames) > 0 {
				info, err := build.FromFiles(filenames).Default().Build()
				if err != nil {
					t.Errorf("build failed: %v", err)
				}
				var buf bytes.Buffer
				inferer := migoinfer.New(info, nil)
				inferer.Raw = false
				inferer.SetOutput(&buf)
				inferer.Analyse()
				if want, got := string(bytes.TrimSpace(migob)), strings.TrimSpace(buf.String()); want != got {
					t.Errorf("Output does not match\nExpect:\n%s\nGot:\n%s\n", want, got)
				}
			} else {
				t.Fail()
			}
		})
	}
}
