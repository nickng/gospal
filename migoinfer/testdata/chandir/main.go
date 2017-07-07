package main

func SendOnly(ch chan<- int) {
	ch <- 1
}

func RecvOnly(ch <-chan int) {
	<-ch
}

func main() {
	ch := make(chan int)
	go SendOnly(ch)
	RecvOnly(ch)
}
