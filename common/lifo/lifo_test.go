package lifo

import (
	"testing"
)

// TestPushAndPop tests basic push and pop operations
func TestPushAndPop(t *testing.T) {
	stack := Stack[int]{}

	// Push items onto the stack
	stack.Push(1)
	stack.Push(2)
	stack.Push(3)

	// Pop and check order (LIFO)
	val, ok := stack.Pop()
	if !ok || val != 3 {
		t.Errorf("Expected 3, got %v", val)
	}

	val, ok = stack.Pop()
	if !ok || val != 2 {
		t.Errorf("Expected 2, got %v", val)
	}

	val, ok = stack.Pop()
	if !ok || val != 1 {
		t.Errorf("Expected 1, got %v", val)
	}

	// Stack should now be empty
	_, ok = stack.Pop()
	if ok {
		t.Errorf("Expected empty stack, but Pop returned a value")
	}
}

// TestPeek tests the Peek operation
func TestPeek(t *testing.T) {
	stack := Stack[string]{}

	stack.Push("A")
	stack.Push("B")

	// Peek should return the last pushed item without removing it
	val, ok := stack.Peek()
	if !ok || val != "B" {
		t.Errorf("Expected B, got %v", val)
	}

	// Peek again to ensure it's still there
	val, ok = stack.Peek()
	if !ok || val != "B" {
		t.Errorf("Expected B again, got %v", val)
	}
}

// TestLen tests the Len() function
func TestLen(t *testing.T) {
	stack := Stack[float64]{}

	if stack.Len() != 0 {
		t.Errorf("Expected length 0, got %d", stack.Len())
	}

	stack.Push(10.5)
	stack.Push(20.3)
	stack.Push(30.7)

	if stack.Len() != 3 {
		t.Errorf("Expected length 3, got %d", stack.Len())
	}

	stack.Pop()
	if stack.Len() != 2 {
		t.Errorf("Expected length 2 after pop, got %d", stack.Len())
	}
}

// TestIsEmpty tests IsEmpty() function
func TestIsEmpty(t *testing.T) {
	stack := Stack[int]{}

	if !stack.IsEmpty() {
		t.Errorf("Expected empty stack, got non-empty")
	}

	stack.Push(42)
	if stack.IsEmpty() {
		t.Errorf("Expected non-empty stack, got empty")
	}

	stack.Pop()
	if !stack.IsEmpty() {
		t.Errorf("Expected empty stack after pop, got non-empty")
	}
}

// TestPopEmpty tests popping from an empty stack
func TestPopEmpty(t *testing.T) {
	stack := Stack[bool]{}

	val, ok := stack.Pop()
	if ok {
		t.Errorf("Expected false for Pop from empty stack, got %v", val)
	}
}

// TestPeekEmpty tests peeking from an empty stack
func TestPeekEmpty(t *testing.T) {
	stack := Stack[rune]{}

	val, ok := stack.Peek()
	if ok {
		t.Errorf("Expected false for Peek from empty stack, got %v", val)
	}
}

// TestCopy tests the Copy() function
func TestCopy(t *testing.T) {
	original := Stack[int]{}
	original.Push(1)
	original.Push(2)
	original.Push(3)

	copied := original.Copy()

	// Ensure the copied stack has the same length
	if copied.Len() != original.Len() {
		t.Errorf("Expected copied stack length %d, got %d", original.Len(), copied.Len())
	}

	// Ensure elements are in the same LIFO order
	for !original.IsEmpty() {
		origVal, _ := original.Pop()
		copyVal, _ := copied.Pop()
		if origVal != copyVal {
			t.Errorf("Copy failed: expected %d, got %d", origVal, copyVal)
		}
	}

	// Ensure copied stack is now empty after popping all elements
	if !copied.IsEmpty() {
		t.Errorf("Expected copied stack to be empty after popping all elements")
	}
}

// TestCopyEmpty tests Copy() on an empty stack
func TestCopyEmpty(t *testing.T) {
	original := Stack[string]{}

	copied := original.Copy()

	if !copied.IsEmpty() {
		t.Errorf("Expected copied stack to be empty, but it's not")
	}
}
