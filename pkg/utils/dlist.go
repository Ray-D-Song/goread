package utils

type DItem[T any] struct {
	Value T
	Prev  *DItem[T]
	Next  *DItem[T]
}

// We only need a head pointer for the double linked list
type DList[T any] struct {
	Head  *DItem[T]
	Slice []T
}

func NewDList[T any]() *DList[T] {
	return &DList[T]{}
}

// This function adds a node to the list after the previous node
// If the previous node is nil, the node will be added to the head of the list
// Returns the next node in the list
func (l *DList[T]) Add(value T, prev *DItem[T]) (next *DItem[T]) {
	if prev == nil {
		l.Head = &DItem[T]{Value: value}
		l.Slice = append(l.Slice, value)
		return l.Head.Next
	}
	prev.Next = &DItem[T]{Value: value}
	prev.Next.Prev = prev
	return prev.Next
}

func (l *DList[T]) Len() int {
	return len(l.Slice)
}
