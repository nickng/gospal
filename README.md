# gospal [![GoDoc](https://godoc.org/github.com/nickng/gospal?status.svg)](http://godoc.org/github.com/nickng/gospal)

## Go Static Program AnaLysing framework

This is a research prototype static analyser for Go programs. Currently the
framework consists of two main tools, `migoinfer` and `ssaview`, but it
should be able to build more backends with different output formats based on this framework.

To build the tool, use `go get`:

```
go get github.com/nickng/gospal/cmd/...
```

### migoinfer

The MiGo infer tool (`cmd/migoinfer`) infers [MiGo
types](http://github.com/nickng/migo) from a Go source code. The formal
definitions of the MiGo types are published in [this paper](http://mrg.doc.ic.ac.uk/publications/fencing-off-go-liveness-and-safety-for-channel-based-programming/), and the format of the output in textual form is described in the [migo](https://github.com/nickng/migo/blob/master/README.md) package.

For example, given this sample program `main.go`,

```
package main

func main() {
	ch := make(chan int)
	go Sender(ch)
	fmt.Println(<-ch)
}

func Sender(ch chan int) {
	ch <- 1
}
```

This is the expected output of the inference, with additional output `def`s that
does not involve communication.

```
$ migoinfer main.go

def main.main():
    let t0 = newchan main.main0.t0_chan0, 0;
    spawn main.Sender(t0);
    recv t0;
def main.Sender(ch):
    send ch
```

This is a research prototype and does not cover all features of Go.
Please report errors with a small fragment of sample code and what you
expect to see, however, noting that it might not be possible to infer the
types soundly due to the limitations of static analysis.

### ssaview

The SSA viewer (`cmd/ssaview`) is a wrapper over the
[`ssa`](http://golang.org/x/tools/go/ssa) package from the extra tools of the Go
project for viewing SSA-form of a given source code. It is similar to
[`ssadump`](https://golang.org/x/tools/cmd/ssadump) but shares the build
configuration with the `migoinfer` tool in this project.
