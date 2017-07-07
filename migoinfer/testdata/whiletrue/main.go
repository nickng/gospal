package main

// This tests for 1 -> 1 block transition in fork (by fork).

func fork(ch chan bool) {
	for {
		b := <-ch
		ch <- b
	}
}

func main() {
	fork1 := make(chan bool)
	fork2 := make(chan bool)
	go fork(fork1)
	go fork(fork2)
	fork1 <- true
	fork2 <- true
}
