package main

func main() {
	close(make(chan struct{}))
}
