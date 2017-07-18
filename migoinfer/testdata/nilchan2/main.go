package main

// This test mimics the behaviour of using a nil context Done channel.

type Context struct {
	done chan struct{}
}

func (ctx *Context) Done() <-chan struct{} {
	return ctx.done
}

func f(c *Context) {
	select {
	case <-c.Done():
	default:
	}
}

func main() {
	c := new(Context)
	f(c)

	d := new(Context)
	d.done = make(chan struct{})
	f(d)
}
