package main

func main() {
	ch := make(chan struct{})
	select {
	case ch <- struct{}{}:
	case <-ch:
	default:
	}
}
