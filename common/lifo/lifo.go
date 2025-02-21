// Package lifo implements lifo stack
package lifo

type Stack[T any] struct {
	items []T
}

// Push adds an item to the stack
func (s *Stack[T]) Push(value T) {
	s.items = append(s.items, value)
}

// Pop removes and returns the last item from the stack
func (s *Stack[T]) Pop() (T, bool) {
	if len(s.items) == 0 {
		var zero T
		return zero, false
	}
	val := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return val, true
}

// Peek returns the last item without removing it
func (s *Stack[T]) Peek() (T, bool) {
	if len(s.items) == 0 {
		var zero T
		return zero, false
	}
	return s.items[len(s.items)-1], true
}

// Len returns the number of items in the stack
func (s *Stack[T]) Len() int {
	return len(s.items)
}

// IsEmpty checks if the stack is empty
func (s *Stack[T]) IsEmpty() bool {
	return len(s.items) == 0
}

// Copy creates a new stack with the same elements
func (s *Stack[T]) Copy() *Stack[T] {
	newStack := &Stack[T]{}
	newStack.items = append([]T{}, s.items...) // Efficient slice copy
	return newStack
}
