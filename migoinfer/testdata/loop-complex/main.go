package main

func main() {
	ch1 := make(chan int)
	ch2 := make(chan int)
	a := 0
	for i := 0; (0 < i || i < 10) && (i == 1 || i == 3 && i == 2); i++ {
		ch1 <- 1
		k := 0
		k++
		for j := 100; j >= 3; j-- {
			a += 1
			<-ch1
		}
		l := 2
		a -= l
		<-ch2
	}
	ch2 <- 2
}
