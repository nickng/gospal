package main

func main() {
	done := make(chan struct{}, 1)
	work := make(chan int, 1)
X:
	for {
		select {
		case <-done:
			break X
		case work <- 1:
			done <- struct{}{}
		}
	}
	<-work
}
