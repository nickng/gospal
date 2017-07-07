package main

type S struct {
	ch chan int
	v  int
}

func set(s *S, v int) {
	s.v = v
}

func main() {
	s := new(S)
	set(s, 10)
	set(s, 10)
}
