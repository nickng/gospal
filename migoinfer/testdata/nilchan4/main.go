package main

func send(x struct{ c chan int }) { x.c <- 1 }
func recv(y struct{ c chan int }) { <-y.c }
func main() {
	x := struct{ c chan int }{}
	go send(x)
	go recv(x)
}
