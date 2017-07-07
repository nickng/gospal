// +build ignore
package main

func main() {
	ch1 := make(chan int, 1)
	ch2 := ch1
	sendAlias(ch1, ch2)
	ch2 = make(chan int, 1)
	sendAlias(ch1, ch2)
	<-ch1
}

func sendAlias(ch1, ch2 chan int) {
	ch2 <- 1
}
