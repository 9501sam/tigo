package utils

import ()

// Stack represents a LIFO (Last In, First Out) stack.
// It uses a slice of interface{} to allow storing elements of any type,
// or you could make it generic if working with Go 1.18+.
type Stack struct {
	elements []interface{}
}

// NewStack creates and returns a new empty Stack.
func NewStack() *Stack {
	return &Stack{
		elements: make([]interface{}, 0), // Initialize with an empty slice
	}
}

// Push adds an element to the top of the stack.
func (s *Stack) Push(item interface{}) {
	s.elements = append(s.elements, item)
}

// Pop removes and returns the top element from the stack.
// It returns nil if the stack is empty. Callers should check for nil.
func (s *Stack) Pop() interface{} {
	if s.IsEmpty() {
		// If the stack is empty, return nil.
		// The caller is responsible for checking if the returned value is nil.
		return nil
	}
	lastIndex := len(s.elements) - 1
	item := s.elements[lastIndex]
	s.elements = s.elements[:lastIndex] // Reslice to remove the last element
	return item
}

// Top returns the top element of the stack without removing it.
// It returns nil if the stack is empty. Callers should check for nil.
func (s *Stack) Top() interface{} {
	if s.IsEmpty() {
		// If the stack is empty, return nil.
		// The caller is responsible for checking if the returned value is nil.
		return nil
	}
	return s.elements[len(s.elements)-1]
}

func (s *Stack) PeekSecond() interface{} {
	if s.IsEmpty() || len(s.elements) < 2 {
		return nil
	}
	return s.elements[len(s.elements)-2]
}

// IsEmpty checks if the stack contains no elements.
func (s *Stack) IsEmpty() bool {
	return len(s.elements) == 0
}

// Size returns the number of elements in the stack.
func (s *Stack) Size() int {
	return len(s.elements)
}

func (s *Stack) GetElements() []interface{} {
	return s.elements
}

// func StackExample() {
// 	myStack := NewStack()
//
// 	fmt.Println("Is stack empty?", myStack.IsEmpty()) // true
// 	fmt.Println("Stack size:", myStack.Size())        // 0
//
// 	myStack.Push("apple")
// 	myStack.Push(123)
// 	myStack.Push(true)
//
// 	fmt.Println("\nAfter pushing elements:")
// 	fmt.Println("Is stack empty?", myStack.IsEmpty()) // false
// 	fmt.Println("Stack size:", myStack.Size())        // 3
//
// 	topElement, err := myStack.Top()
// 	if err == nil {
// 		fmt.Println("Top element (peek):", topElement) // true
// 	}
//
// 	poppedElement, err := myStack.Pop()
// 	if err == nil {
// 		fmt.Println("Popped element:", poppedElement) // true
// 	}
// 	fmt.Println("Stack size after pop:", myStack.Size()) // 2
//
// 	poppedElement, err = myStack.Pop()
// 	if err == nil {
// 		fmt.Println("Popped element:", poppedElement) // 123
// 	}
//
// 	poppedElement, err = myStack.Pop()
// 	if err == nil {
// 		fmt.Println("Popped element:", poppedElement) // apple
// 	}
//
// 	fmt.Println("Stack size after all pops:", myStack.Size()) // 0
// 	_, err = myStack.Pop()
// 	if err != nil {
// 		fmt.Println("Error when popping from empty stack:", err) // Error message
// 	}
// }
