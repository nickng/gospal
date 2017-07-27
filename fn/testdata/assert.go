// +build ignore

package main

type u struct {
}

func (*u) f() {}
func (*u) g() {}
func (*u) z() {}

type A interface {
	f()
	g()
}

type B interface {
	f()
}

func main() {
	var x A
	x = new(u)
	x.(A).(B).f()
}
