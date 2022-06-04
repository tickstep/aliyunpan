package collection

import (
	"fmt"
	"testing"
)

type item struct {
	Name string
}

func (i *item) HashCode() string {
	return i.Name
}

func TestRemove(t *testing.T) {
	q := NewFifoQueue()
	q.Push(&item{Name: "1"})
	q.Push(&item{Name: "2"})
	q.Push(&item{Name: "3"})
	q.Push(&item{Name: "4"})
	q.Remove(&item{Name: "3"})
	fmt.Println(q)
}
