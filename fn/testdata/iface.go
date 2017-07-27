// +build ignore

package main

type t struct{}

func (*t) f() {}

type fer interface {
	f()
}

func main() {
	var x fer
	x = new(t)
	x.f()
}
