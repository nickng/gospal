package main

type S struct {
	ch chan int // nilchan
	v  int
}

func set(s *S, v int) {
	s.v = v
}

func main() {
	s := new(S)
	set(s, 10)
	set(s, 10)
	<-s.ch

	n := new(S)
	set(n, 11)
	n.ch = make(chan int)
	<-n.ch
}
