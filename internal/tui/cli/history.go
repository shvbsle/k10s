package cli

import (
	"github.com/gammazero/deque"
)

type History interface {
	Push(string)
	MoveIndex(int) string
	ResetIndex()
}

func NewCommandHistory(cap int) *commandHistory {
	return &commandHistory{
		cap:   cap,
		index: -1,
		deque: deque.Deque[string]{},
	}
}

type commandHistory struct {
	cap   int
	index int
	deque deque.Deque[string]
}

func (p *commandHistory) ResetIndex() {
	p.index = -1
}

func (p *commandHistory) Push(item string) {
	p.deque.PushFront(item)
	for p.deque.Len() > p.cap {
		p.deque.IterPopBack()
	}
}

// MoveIndex moves the index and returns the history item at the position.
func (p *commandHistory) MoveIndex(amount int) string {
	if p.deque.Len() == 0 {
		return ""
	}
	p.index = max(min(p.index+amount, p.deque.Len()-1), -1)
	if p.index < 0 {
		return ""
	}
	return p.deque.At(p.index % p.deque.Len())
}
