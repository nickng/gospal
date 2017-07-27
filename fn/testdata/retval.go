// +build ignore

package main

type t struct{}

func New() *t { return new(t) }
func (*t) f() {}

type A interface {
	f()
}

func main() {
	var x A
	x = New()
	x.f()
}
