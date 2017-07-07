package main

func main() {
	ch := make(chan struct{})
	select {
	case <-ch:
	case ch <- struct{}{}:
	}
}
