package loop

import (
	"errors"
	"sync"
)

var ErrEmptyStack = errors.New("error: empty stack")

// Stack is a stack of ssa.BasicBlock
type Stack struct {
	sync.Mutex
	s []*Info
}

// NewStack creates a new LoopStack.
func NewStack() *Stack {
	return &Stack{s: []*Info{}}
}

// Push adds a new Info to the top of stack.
func (s *Stack) Push(i *Info) {
	s.Lock()
	defer s.Unlock()
	s.s = append(s.s, i)
}

// Pop removes a Loop from top of stack.
func (s *Stack) Pop() (*Info, error) {
	s.Lock()
	defer s.Unlock()

	size := len(s.s)
	if size == 0 {
		return nil, ErrEmptyStack
	}
	l := s.s[size-1]
	s.s = s.s[:size-1]
	return l, nil
}

// IsEmpty returns true if stack is empty.
func (s *Stack) IsEmpty() bool {
	return len(s.s) == 0
}
