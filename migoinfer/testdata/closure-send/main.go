package main

func main() {
	val := struct{}{}
	ch := make(chan struct{})
	go func() {
		ch <- val
	}()
	<-ch
}
