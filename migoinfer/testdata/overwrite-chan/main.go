package main

func main() {
	ch0 := make(chan int)
	from := ch0
	to := make(chan int)
	go forward(from, to)
	for i := 0; i < 1; i++ {
		from = to
		to = make(chan int)
		go forward(from, to)
	}
	go func(ch0 chan int) { ch0 <- 1 }(ch0)
	print(<-to)
}

func forward(from, to chan int) {
	v := <-from
	print(v)
	to <- v
}
