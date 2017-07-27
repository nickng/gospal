// +build ignore

package main

type t struct{}

func New() *t         { return new(t) }
func (a *t) f()       {}
func (a *t) chain() A { return a }

type A interface {
	chain() A
}

func main() {
	var x A
	x = New()
	x.chain().chain()
}
