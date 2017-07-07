package main

// Context tracking test.
// Instances of two function calls should be different.

type T struct{ ch chan int }

func newch() chan int {
	return make(chan int, 1)
}

func setch(t *T) {
	t.ch = make(chan int, 1)
}

func main() {
	x := newch() // newch_1
	y := newch() // newch_2
	y <- 1
	x <- <-y
	a := &T{}
	b := new(T)
	setch(a) // setch_1
	setch(b) // setch_2
	b.ch <- 1
	a.ch <- <-b.ch
}
