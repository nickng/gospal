package main

// Sender sends.
type Sender interface {
	Send(int)
}

// Receiver receives.
type Receiver interface {
	Recv() int
}

// sr is a struct that stores a channel for sending and receiving.
type sr struct {
	ch chan int
}

func (sr sr) Send(v int) {
	sr.ch <- v
}

func (sr sr) Recv() int {
	return <-sr.ch
}

func main() {
	var x Sender = sr{ch: make(chan int, 1)}
	var y Receiver = x.(sr)
	x.Send(1)
	x.(Receiver).Recv()
	y.Recv()

	go x.Send(1)
	go y.Recv()
	x.Send(1)
	y.Recv()
}
